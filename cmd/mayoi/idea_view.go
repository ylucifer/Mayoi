package main

// This file is ours alone: upstream never touches it, so merges with
// upstream pass it by. The IDEA look's own views live here; the views
// they replace keep upstream's bodies in view.go and sidebar.go, with
// an `if ideaLayout` branch at their head that hands the work over.
// When upstream changes one of those bodies, this file's copy is re-read
// and re-pasted, as docs/IDEA_PLAN.md §7 tells.

import (
	"fmt"
	"math"
	"path/filepath"
	"runtime"

	"github.com/egoist/mygo/ui"

	"mayoi/internal/remote"
	"mayoi/locales"
)

// view builds the window the IDEA new UI way: a main toolbar, a tool
// window stripe at each side, the sidebar docking left or right, and a
// taller status bar. It follows upstream's view step by step, past the
// layout it lays out.
func (a *app) ideaView(c *ui.Context) {
	// The window's, which may be another than the frame before's as the
	// project moves to another window.
	a.setServices(c.Services())
	a.review.ctx = c
	// Restoring a preview uses font metrics to recover its scroll position.
	if p := a.settings.Prefs; !a.font.is(p) {
		a.font = newCodeFont(p)
	}
	if !a.started {
		// Before what was posted meanwhile, as opening the file the app was
		// started with: it opens in the columns of the last session.
		a.started = true
		a.start()
	}
	a.drain()
	a.pumpChain()
	// The key the keymap took is let go of where its release does not
	// come: in a place with no text, or another window.
	if a.held != nil && (a.blurred || a.keyFocus != placeEditor && a.keyFocus != placeInput) {
		a.held = nil
	}
	// The places that take the keys say so as they are built.
	a.keyFocus = placeNone
	a.pal = a.applyTheme(c)
	updateLinuxControlsTheme(a.fade.to)
	a.themeTerminals()
	a.fontTerminals()
	a.copyTerminals()
	p := a.pal
	a.builds++
	if now := c.Now(); now != a.frameAt {
		a.frameAt = now
		a.frame++
		a.dragging, a.dragNow = a.dragNow, false
	}
	a.shortcuts(c)
	c.SetZoom(float32(math.Pow(1.2, float64(a.settings.ZoomLevel))))
	// Which pane has the keyboard focus, as the frame before left it, and
	// which column: its editor is the one typing is of.
	for i, col := range a.cols {
		switch t := col.tab(); {
		case t == nil:
		case t.ed != nil && t.ed.vscroll.FocusWithin(c):
			a.focus, a.col = paneEditor, i
			t.ed.typist = t.ed
		case t.term != nil && (t.term.view.FocusWithin(c) || t.term.findInput.Focused(c)):
			a.focus, a.col = paneEditor, i
		case t.view != nil && t.view.pane.FocusWithin(c):
			a.focus, a.col = paneEditor, i
		case t.diff != nil && t.diff.pane.FocusWithin(c):
			a.focus, a.col = paneEditor, i
		case t.ed != nil && t.ed.preview != nil && t.ed.preview.pane.FocusWithin(c):
			a.focus, a.col = paneEditor, i
		}
	}
	a.autoSaveTick()
	switch {
	case a.panelTerm != nil && (a.panelTerm.view.FocusWithin(c) || a.panelTerm.findInput.Focused(c)):
		a.focus = paneTerminal
	case a.files.List.FocusWithin(c):
		a.focus = paneFiles
	}

	ui.Column(c).Fill().Background(p.chrome).Children(func() {
		a.ideaTitleBar(c)
		// The panes keep their places in the tree, by keys, as the sidebar
		// and the panel come and go, so that the editors keep their state.
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			s, w := a.settings, a.window
			// The tool window stripes are always at the window's edges,
			// as IDEA's are, whatever shows between them.
			a.leftStripe(c)
			if a.rail != nil {
				a.rail(c)
			}
			// The sidebar docks left, as the explorer does, and right
			// while it shows the source control, as IDEA's Git window
			// does; a user may keep every view at the right.
			sbView := w.SidebarView
			if a.root == "" {
				sbView = ""
			}
			sbRight := s.SidebarRight || sbView == "git"
			// The panel is at the window's edge, past the sidebar, so
			// that with both on a side the panel is the outer one.
			side := ""
			if w.PanelOpen {
				side = a.panelSide()
			}
			sidebar := func() {
				if w.SidebarClosed {
					return
				}
				// Its splitter is along the edge facing the editors.
				at, dir := edgeRight, float32(1)
				box := ui.Column(c.Key("sidebar")).Width(w.SidebarWidth).Shrink(0).BorderColor(p.seam())
				a.grained(box)
				if sbRight {
					at, dir = edgeLeft, -1
					box.BorderWidth(0, 0, 0, 1)
				} else {
					box.BorderWidth(0, 1, 0, 0)
				}
				box.Children(func() {
					a.sidebar(c)
					a.splitter(c, at, func(d float32) {
						width, _ := c.Size()
						w.SidebarWidth = clamp(w.SidebarWidth+dir*d, 160, max(width-320, 160))
					})
				})
			}
			if side == "left" {
				a.sidePanel(c, false)
			}
			if !sbRight {
				sidebar()
			}
			main := ui.Column(c.Key("main")).MinWidth(0)
			if (side == "left" || side == "right") && a.panelMax {
				// A panel maximized at a side takes the row, so the
				// editors give it theirs rather than sharing it. One at
				// the bottom is in the editors' column, and takes theirs.
				main.Grow(0).Width(0)
			} else {
				main.Grow(1)
				// The stripes hold the window's edges, so the card never
				// meets one: it keeps its border and shadow alone.
				a.asCard(main, false, false)
			}
			main.Children(func() { a.main(c) })
			if sbRight {
				sidebar()
			}
			if side == "right" {
				a.sidePanel(c, true)
			}
			a.rightStripe(c)
		})
		a.statusBar(c)
	})
	a.quickOpenView(c)
	a.branchPickerView(c)
	a.stashPickerView(c)
	a.prPickerView(c)
	a.prChecksView(c)
	a.worktreePickerView(c)
	a.repositoryPickerView(c)
	a.recentPickerView(c)
	a.themePickerView(c)
	a.remotePickerView(c)
	a.conflictsView(c)
	a.diagramOverlayView(c)
	a.promptView(c)
	a.confirmView(c)
	a.signinView(c)
	a.windowKeys(c)
	if inEditor := a.keyFocus == placeEditor; inEditor != a.cutInEditor {
		a.cutInEditor = inEditor
		a.host.menuChanged()
	}
	for _, msg := range a.errors {
		c.Toast(msg)
	}
	a.errors = a.errors[:0]
	if len(a.notes) > 0 {
		// The character holds up its megaphone for a while.
		a.toldAt = c.Now()
	}
	for _, n := range a.notes {
		if n.action != nil {
			c.ToastAction(n.message, n.label, n.action)
		} else {
			c.Toast(n.message)
		}
	}
	a.notes = a.notes[:0]
	a.tick()
}

// ideaTitleBar is the main toolbar, as IDEA's new UI's is: the menu,
// the project and its branch at the left, a search box at the middle,
// and the actions used most at the right.
func (a *app) ideaTitleBar(c *ui.Context) {
	p := a.pal
	bar := c.TitleBar()
	rightPad := bar.Right + 8
	if runtime.GOOS == "linux" && bar.Right > 0 {
		rightPad = bar.Right - 2
	}
	row := ui.Row(c).Height(max(bar.Height, titleH())).Padding(0, rightPad, 0, bar.Left+10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(p.hairline()).DragWindow()
	a.grained(row)
	row.Children(func() {
		// The left: the menu, and the project with its branch, as
		// IDEA's VCS control is.
		ui.Row(c).Basis(0).Grow(1).MinWidth(56).Gap(8).AlignItems(ui.Center).Children(func() {
			if hostOS != "darwin" {
				a.iconButton(c, iconMenu, locales.Menu(), false).Menu(a.appUIMenu)
			}
			a.vcsControl(c)
		})
		// The middle: the search box.
		ui.Row(c).Width(150).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			a.ideaCommandCenter(c)
		})
		// The right: the actions used most.
		ui.Row(c).Basis(0).Grow(1).MinWidth(56).Justify(ui.End).Gap(8).AlignItems(ui.Center).Children(func() {
			a.toolButton(c, iconCommand, a.withKey(locales.CmdGoCommands(), "go.commands"), false, func() { a.openQuick(">") })
			a.toolButton(c, iconSettings, a.withKey(locales.CmdFileSettings(), "file.settings"), false, func() { a.host.showSettings("", "") })
			a.toolButton(c, iconSparkles, a.withKey(locales.CmdViewSelectTheme(), "view.selectTheme"), false, a.openThemePicker)
		})
	})
}

// vcsControl is the project's name and the branch shown at the left of
// the main toolbar, as IDEA's VCS control is: the branch opens its
// picker.
func (a *app) vcsControl(c *ui.Context) {
	p := a.pal
	if a.root == "" {
		return
	}
	name := filepath.Base(a.root)
	if host := remote.HostOf(a.root); host != "" {
		name = filepath.Base(hostPath(a.root))
	}
	if len(a.repositories.contexts) > 1 {
		name = a.repositoryLabel()
	}
	ui.Text(c, name).FontSize(13).FontWeight(600).TextColor(p.text).SingleLine().Shrink(1)
	repo := a.repository()
	if b := repo.branch(); b != "" {
		bb := ui.ButtonBase(c.Key("toolbar:branch")).Height(24).Padding(0, 8).Gap(6).Radius(6).
			AlignItems(ui.Center).Label(locales.BranchLabel(b)).Tooltip(locales.BranchStatusTip(b)).Disabled(repo.git.busy != "")
		if bb.Hovered() {
			bb.Background(p.hover)
		}
		bb.Transition(hoverFade)
		bb.Children(func() {
			col := p.muted
			if bb.Hovered() {
				col = p.text
			}
			ui.Icon(c, iconBranch).FontSize(13).TextColor(col)
			ui.Text(c, b).FontSize(12).TextColor(col).SingleLine().Shrink(1)
		})
		if bb.Clicked() {
			repo.openBranchPicker()
		}
	}
}

// toolButton is a button of the main toolbar: a square of 32 points
// with a 16-point icon, a little larger than the ones of the panels, as
// IDEA's toolbar buttons are.
func (a *app) toolButton(c *ui.Context, icon *ui.SVG, label string, on bool, fn func()) {
	p := a.pal
	b := ui.ButtonBase(c).Size(32, 32).Radius(6).Justify(ui.Center).AlignItems(ui.Center).Label(label).Tooltip(label)
	col := p.muted
	switch {
	case b.Pressed():
		b.Background(p.hover)
		col = p.text
	case b.Hovered():
		b.Background(p.hover)
		col = p.text
	}
	if on {
		col = p.text
	}
	b.Transition(revealFade)
	pressPose(b)
	b.Children(func() { ui.Icon(c, icon).FontSize(16).TextColor(col) })
	if fn != nil && b.Clicked() {
		fn()
	}
}

// stripeButton is a tool window's button on a stripe at the window's
// left or right: a rounded button of its own, a 20-point icon at its
// middle, and while the tool window shows a highlight behind it, as
// IDEA's stripes light their buttons.
func (a *app) stripeButton(c *ui.Context, key string, icon *ui.SVG, label string, on bool, count int, fn func()) {
	p := a.pal
	b := ui.ButtonBase(c.Key(key)).Size(stripeW, stripeBtnH).Justify(ui.Center).AlignItems(ui.Center).Label(label).Tooltip(label)
	col := p.muted
	if on || b.Hovered() {
		col = p.text
	}
	b.Transition(hoverFade)
	pressPose(b)
	b.Children(func() {
		if on {
			ui.Box(c.Key("on")).Absolute().Left(3).Right(3).Top(3).Bottom(3).Radius(6).Background(p.active)
		} else if b.Hovered() {
			ui.Box(c.Key("over")).Absolute().Left(3).Right(3).Top(3).Bottom(3).Radius(6).Background(p.hover)
		}
		ui.Icon(c, icon).FontSize(20).TextColor(col)
		if count > 0 {
			label := fmt.Sprint(count)
			if count > 99 {
				label = "99+"
			}
			// The count rolls to a new one, as a counter's digits do.
			ui.Text(c.Key("count"), label).Absolute().Top(2).Right(2).FontSize(9).FontWeight(600).Padding(0, 3).Radius(6).
				MinWidth(13).TextAlign(ui.Center).Background(p.fill).TextColor(p.fillText).
				ContentTransition(ui.ContentTransition{Numeric: true}).Transition(dotPop)
		}
	})
	if b.Clicked() {
		fn()
	}
}

// stripeViews lays a stripe's tool window buttons out: the views of the
// tree above, and the panel's entry below a divider at the foot, as
// IDEA's stripes are. At the right the source control takes the tree
// views' place, for its Git window docks there.
func (a *app) stripeViews(c *ui.Context, right bool) {
	w := a.window
	view := w.SidebarView
	if a.root == "" {
		view = ""
	}
	open := !w.SidebarClosed
	// The panel's entry is on the stripe of the side it docks at, and
	// at the left when it is at the bottom.
	panelHere := right == (a.panelSide() == "right")
	toggleView := func(on bool, show func()) {
		if on && open {
			a.toggleSidebar()
			return
		}
		show()
	}
	if a.root != "" {
		if right {
			a.stripeButton(c, "stripe:scm", iconBranch, a.withKey(locales.SidebarScm(), "view.scm"), view == "git" && open, a.changeCount(), func() {
				toggleView(view == "git", a.showSourceControl)
			})
		} else {
			a.stripeButton(c, "stripe:explorer", iconFiles, a.withKey(locales.SidebarExplorer(), "view.explorer"), view == "" && open, 0, func() {
				toggleView(view == "", a.showExplorer)
			})
			a.stripeButton(c, "stripe:search", iconSearch, a.withKey(locales.SidebarSearch(), "edit.findInFiles"), view == "search" && open, 0, func() {
				toggleView(view == "search", func() { a.showSearch(false) })
			})
		}
	}
	if !panelHere {
		return
	}
	// The lower segment at the foot, past a divider.
	ui.Column(c).Grow(1)
	ui.Box(c).Shrink(0).FillWidth().Height(1).Margin(0, 8).Background(a.pal.hairline())
	a.stripeButton(c, "stripe:terminal", iconTerminal, a.withKey(locales.CmdViewTerminal(), "view.terminal"), w.PanelOpen, 0, a.togglePanel)
}

// leftStripe is the tool window stripe at the window's left: the
// explorer's and search's views above, the panel's below.
func (a *app) leftStripe(c *ui.Context) {
	p := a.pal
	stripe := ui.Column(c.Key("stripe:left")).Width(stripeW).Shrink(0).Padding(6, 0).Background(p.chrome).
		BorderWidth(0, 1, 0, 0).BorderColor(p.hairline())
	a.grained(stripe)
	stripe.Children(func() { a.stripeViews(c, false) })
}

// rightStripe is the tool window stripe at the window's right: the
// source control's view above, the panel's below when it docks there.
func (a *app) rightStripe(c *ui.Context) {
	p := a.pal
	stripe := ui.Column(c.Key("stripe:right")).Width(stripeW).Shrink(0).Padding(6, 0).Background(p.chrome).
		BorderWidth(0, 0, 0, 1).BorderColor(p.hairline())
	a.grained(stripe)
	stripe.Children(func() { a.stripeViews(c, true) })
}

// ideaCommandCenter is the box in the middle of the main toolbar: the
// search box, as IDEA's is, which opens quick open.
func (a *app) ideaCommandCenter(c *ui.Context) {
	p := a.pal
	b := ui.ButtonBase(c.Key("command-center")).Height(30).FillWidth().Padding(0, 12).Gap(8).
		Radius(6).Justify(ui.Center).AlignItems(ui.Center).Border(1, p.border).Label(locales.QuickOpenLabel()).Tooltip(a.withKey(locales.QuickOpenLabel(), "go.quickOpen"))
	b.Background(p.surface)
	hovered := b.Hovered()
	switch {
	case b.Pressed():
		b.Background(p.surfacePressed)
	case hovered:
		b.Background(p.surfaceHover).Border(1, mix(p.border, p.muted, 0.3))
	}
	b.Transition(hoverFade)
	b.Children(func() {
		col := p.faint
		if hovered {
			col = p.muted
		}
		ui.Icon(c, iconSearch).FontSize(16).TextColor(col)
		ui.Text(c, locales.QuickOpenLabel()).FontSize(13).TextColor(p.faint).SingleLine()
	})
	if b.Clicked() {
		a.openQuick("")
	}
}

// ideaSidebarHeader is the tool window's header, as IDEA's is: its
// title at the left and its actions at the right; the views it may show
// are the stripe's buttons at the window's edge.
func (a *app) ideaSidebarHeader(c *ui.Context, scm, search, hovered bool) {
	p := a.pal
	ui.Row(c).Height(36).Padding(0, 6, 0, 12).Gap(2).AlignItems(ui.Center).Children(func() {
		title, short, actions := locales.SidebarTitleExplorer(), "", 0
		switch {
		case scm:
			title, short = locales.SidebarTitleScm(), locales.SidebarTitleScmShort()
			if a.repository().git.repo != nil {
				actions = 1
			}
		case search:
			title = locales.SidebarTitleSearch()
			if hovered {
				actions = 3
			}
		}
		// The gaps, the paddings and the actions' buttons take the
		// rest of the row.
		room := a.window.SidebarWidth - 1 - 12 - 6 - float32(actions)*28
		size, spacing := p.titleLook(13, 0)
		title = fitTitle(c, p.title(title), p.title(short), room, size, spacing)
		ui.Text(c, title).FontSize(size).FontWeight(600).LetterSpacing(spacing).TextColor(p.text).Grow(1).
			Padding(0, 0, 0, 0).SingleLine()
		if scm {
			a.scmTitleActions(c)
			return
		}
		if search {
			a.searchTitleActions(c, hovered)
		}
	})
}
