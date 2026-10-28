package app

import (
	"fmt"

	"afvi/teach-imgui/internal/updater"
	"afvi/teach-imgui/internal/version"

	"github.com/AllenDang/cimgui-go/imgui"
)

// drawUpdateDialog is the update window: what the server offers, how far the download is and
// the two buttons that matter (install now / later). It is a popup modal like About, so it
// cannot be lost behind the viewer, and it is only opened when there is something to say or
// when the user asked from the menu.
func (a *App) drawUpdateDialog() {
	u := &a.update
	if u.open {
		imgui.OpenPopupStr("update")
		u.open = false
	}
	vp := imgui.MainViewport()
	center := imgui.NewVec2(vp.Pos().X+vp.Size().X/2, vp.Pos().Y+vp.Size().Y/2)
	imgui.SetNextWindowPosV(center, imgui.CondAlways, imgui.NewVec2(0.5, 0.5))
	imgui.SetNextWindowSizeV(imgui.NewVec2(sc(470), sc(260)), imgui.CondAlways)
	flags := imgui.WindowFlagsNoTitleBar | imgui.WindowFlagsNoResize | imgui.WindowFlagsNoMove |
		imgui.WindowFlagsNoSavedSettings | imgui.WindowFlagsNoScrollbar
	imgui.PushStyleVarVec2(imgui.StyleVarWindowPadding, imgui.NewVec2(0, 0))
	open := imgui.BeginPopupModalV("update", nil, flags)
	imgui.PopStyleVar()
	if !open {
		return
	}
	if u.closeReq {
		u.closeReq = false
		imgui.CloseCurrentPopup()
	}
	wp, ws := imgui.WindowPos(), imgui.WindowSize()
	dl := imgui.WindowDrawList()
	DrawEdge(dl, wp, imgui.NewVec2(wp.X+ws.X, wp.Y+ws.Y), EdgeWindow)
	Caption(dl, imgui.NewVec2(wp.X+3, wp.Y+3), imgui.NewVec2(wp.X+ws.X-3, wp.Y+3+sc(20)), T("update.title"), true)

	x := wp.X + sc(16)
	y := wp.Y + sc(40)
	imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
	imgui.TextUnformatted(T("update.installed", "version", version.String()))
	y += sc(20)

	switch {
	case u.ready:
		imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
		imgui.TextUnformatted(T("update.readyLong", "version", u.offer.VersionName))
	case u.offer != nil:
		imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
		imgui.TextUnformatted(T("update.availableLong", "version", u.offer.VersionName))
	default:
		imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
		imgui.TextUnformatted(a.updateSummary())
	}
	y += sc(24)

	// progress or release notes
	if u.downloading {
		total := max64(u.dlTotal.Load(), 1)
		frac := float32(u.dlDone.Load()) / float32(total)
		imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
		imgui.ProgressBarV(frac, imgui.NewVec2(ws.X-sc(32), sc(16)), "")
		y += sc(22)
		imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
		imgui.TextUnformatted(T("update.progress",
			"done", humanBytes(u.dlDone.Load()), "total", humanBytes(total),
			"pct", int(100*frac)))
		y += sc(20)
	} else if u.offer != nil {
		if u.offer.Mandatory {
			imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
			imgui.TextColored(CurrentPalette().Warn, T("update.mandatory"))
			y += sc(20)
		}
		imgui.SetCursorScreenPos(imgui.NewVec2(x, y))
		imgui.PushTextWrapPosV(ws.X - sc(16))
		imgui.TextUnformatted(notesFor(u.offer))
		imgui.PopTextWrapPos()
	}
	if u.dlErr != "" {
		imgui.SetCursorScreenPos(imgui.NewVec2(x, y+sc(60)))
		imgui.PushTextWrapPosV(ws.X - sc(16))
		imgui.TextColored(CurrentPalette().NG, T("update.err.download", "msg", u.dlErr))
		imgui.PopTextWrapPos()
	}

	// buttons: the primary action depends on where we are
	btnY := wp.Y + ws.Y - sc(32)
	var labels []struct {
		id, text string
		primary  bool
	}
	switch {
	case u.installing:
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-wait", T("update.installing"), true})
	case u.ready:
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-install", T("update.install"), true})
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-later", T("update.later"), false})
	case u.downloading:
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-later", T("update.hide"), false})
	case u.off || u.client == nil:
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-close", T("cfg.ok"), true})
	case u.checking:
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-later", T("update.hide"), false})
	default:
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-check", T("update.checkNow"), u.offer == nil})
		if u.offer != nil {
			labels = append(labels, struct {
				id, text string
				primary  bool
			}{"update-ignore", T("update.ignore"), false})
		}
		labels = append(labels, struct {
			id, text string
			primary  bool
		}{"update-close", T("cfg.ok"), false})
	}

	// right-aligned row, primary on the right
	bx := wp.X + ws.X - sc(16)
	for i := len(labels) - 1; i >= 0; i-- {
		l := labels[i]
		w := ButtonWidth(l.text)
		if w < sc(80) {
			w = sc(80)
		}
		bx -= w
		imgui.SetCursorScreenPos(imgui.NewVec2(bx, btnY))
		if ButtonSized(l.text, l.id, imgui.NewVec2(w, imgui.FrameHeight()+2), l.primary) {
			a.updateButton(l.id)
		}
		bx -= sc(8)
	}

	if imgui.IsKeyPressedBool(imgui.KeyEscape) && !u.installing {
		imgui.CloseCurrentPopup()
	}
	imgui.EndPopup()
}

// updateButton handles one button of the update dialog.
func (a *App) updateButton(id string) {
	u := &a.update
	switch id {
	case "update-install":
		a.requestInstall()
	case "update-later", "update-close", "update-hide":
		imgui.CloseCurrentPopup()
	case "update-check":
		a.startUpdateCheck(true)
	case "update-ignore":
		if u.offer != nil {
			u.ignoredFor = u.offer.VersionCode
		}
		imgui.CloseCurrentPopup()
	}
}

// notesFor is the release note line of an offer, trimmed for the dialog.
func notesFor(u *updater.Update) string {
	if u.Notes == "" {
		return T("update.nonotes")
	}
	return u.Notes
}

// humanBytes formats a byte count for the progress line.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
