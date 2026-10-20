AFVI Teach build server - everything the automatic build keeps lives in this folder.
(written by deploy/server/install.sh, overwritten when it runs again; details: docs/SERVER_CI.md in the repo)

  bin/ci              the driver. `bin/ci` alone lists the commands, `bin/ci status` shows the state.
  etc/ci.conf         settings (repository, retention, time limit). Never overwritten by an update.
  etc/github-token    GitHub token for publishing, mode 600. Written by `bin/ci set-token`.

  out/                RESULTS. One folder per build, nothing else in it:
    main/<date>-<sha>/    every build of the main branch (the newest 10 are kept)
    tags/<tag>/           every v* tag, kept for ever
    latest-main  ->       newest main build            latest-release ->  newest tag build
    each folder holds:    AFVI-Teach-Setup-<version>.exe, SHA256SUMS.txt, build.log, meta.json

  src.git/            bare mirror of the GitHub repo (fetch only, safe to delete: it is cloned again)
  cache/              Go module cache and build cache (safe to delete, the next build is just slower)
  work/               throwaway checkout of a running build, removed when it ends
  state/lock          one build at a time

Scheduling: systemd user timer afvi-ci.timer (every minute, ~/.config/systemd/user/).
  systemctl --user list-timers afvi-ci.timer       when it runs next
  journalctl --user -u afvi-ci -n 50               what the last runs did
  systemctl --user stop afvi-ci.timer              pause the automatic build
