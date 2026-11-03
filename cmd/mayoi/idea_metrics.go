package main

// This file is ours alone: upstream never touches it, so merges with
// upstream pass it by. It gathers the metrics of the IDEA look, as
// idea_view.go gathers its layout, so that the numbers IDEA's new UI
// lives by sit in one place and the upstream files only refer to them.

// ideaLayout marks the IDEA look as the app's own. The upstream look is
// kept behind it, so a glance at a view's head shows which way it goes,
// and the day we want the old look back it is one constant away.
const ideaLayout = true

const (
	// ideaTitleBarH is the height of the main toolbar, which IDEA's is:
	// the menu, the project and its branch at the left, a search box at
	// the middle, the actions used most at the right.
	ideaTitleBarH = 40
	// ideaTabStripH is the height of the tabs' strip, a little taller
	// than upstream's, to match the toolbar.
	ideaTabStripH = 38
	// ideaStatusBarH is the height of the status bar, as IDEA's is.
	ideaStatusBarH = 30
	// stripeW is the width of the tool window stripes at the window's
	// left and right, which hold the tool windows' buttons, stripeBtnH
	// the height of one of the buttons.
	stripeW    = 44
	stripeBtnH = 32
)

// titleH is the height of the main toolbar, IDEA's when ideaLayout,
// upstream's title bar otherwise.
func titleH() float32 {
	if ideaLayout {
		return ideaTitleBarH
	}
	return titleBarH
}

// tabH is the height of the tabs' strip.
func tabH() float32 {
	if ideaLayout {
		return ideaTabStripH
	}
	return tabStripH
}

// statusH is the height of the status bar.
func statusH() float32 {
	if ideaLayout {
		return ideaStatusBarH
	}
	return statusBarH
}

// statusItemH is the height of a button on the status bar: it keeps a
// little of the bar's height above and below itself.
func statusItemH() float32 {
	if ideaLayout {
		return ideaStatusBarH - 4
	}
	return statusBarH - 6
}

// statusTextSize is the size of the status bar's text, which IDEA's
// larger bar takes larger.
func statusTextSize() float32 {
	if ideaLayout {
		return 13
	}
	return 11.5
}

// statusPad is the status bar's padding at its ends, statusGap the room
// between its items.
func statusPad() float32 {
	if ideaLayout {
		return 10
	}
	return 6
}

func statusGap() float32 {
	if ideaLayout {
		return 4
	}
	return 2
}
