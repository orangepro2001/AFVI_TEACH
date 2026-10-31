package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// stdioDrainTimeout is how long RunStdio waits for running tool calls after
// stdin was closed.
const stdioDrainTimeout = 90 * time.Second

// RunStdio serves MCP over stdin/stdout: one JSON-RPC message per line. stdout is
// reserved for protocol traffic; all logging goes to the logger (stderr or a
// file).
func RunStdio(ctx context.Context, s *Server, logger *log.Logger) error {
	return runStdio(ctx, os.Stdin, os.Stdout, s, logger)
}

func runStdio(ctx context.Context, stdin io.Reader, stdout io.Writer, s *Server, logger *log.Logger) error {
	in := bufio.NewReaderSize(stdin, 1<<20)
	out := bufio.NewWriterSize(stdout, 1<<16)

	var writeMu sync.Mutex
	var inflight sync.WaitGroup

	write := func(payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		if _, err := out.Write(payload); err != nil {
			return err
		}
		if err := out.WriteByte('\n'); err != nil {
			return err
		}
		return out.Flush()
	}

	logger.Printf("stdio 模式就绪（stdout 只写 JSON-RPC）")

	// Reads happen in their own goroutine so SIGINT/SIGTERM (ctx) can stop the
	// loop - a blocking read on stdin would otherwise swallow the signal.
	type incoming struct {
		msg []byte
		err error
	}
	msgs := make(chan incoming)
	go func() {
		for {
			msg, err := readMessage(in)
			select {
			case msgs <- incoming{msg: msg, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()

	drain := func() {
		done := make(chan struct{})
		go func() { inflight.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(stdioDrainTimeout):
			logger.Printf("仍有请求在执行，%s 后退出", stdioDrainTimeout)
		}
	}

	for {
		var item incoming
		select {
		case <-ctx.Done():
			logger.Printf("收到退出信号，等待在途请求")
			drain()
			return nil
		case item = <-msgs:
		}
		if len(item.msg) > 0 {
			inflight.Add(1)
			go func(msg []byte) {
				defer inflight.Done()
				resp, ok := s.HandleEmit(ctx, msg, write)
				if !ok {
					return
				}
				_ = write(resp)
			}(item.msg)
		}
		if item.err != nil {
			if errors.Is(item.err, io.EOF) {
				// stdin closed: no new requests, but finish the ones in flight so an
				// answer is not lost (a well-behaved client waits for it too).
				logger.Printf("stdin 已关闭，等待在途请求")
				drain()
				return nil
			}
			return item.err
		}
	}
}

// readMessage reads one newline-delimited JSON-RPC message, skipping blank lines.
func readMessage(r *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		chunk, err := r.ReadBytes('\n')
		buf = append(buf, chunk...)
		if trimmed := bytes.TrimSpace(buf); len(trimmed) > 0 {
			return trimmed, err
		}
		if err != nil {
			return nil, err
		}
		buf = buf[:0]
	}
}
