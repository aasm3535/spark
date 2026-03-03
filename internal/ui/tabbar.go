package ui

import (
	"gioui.org/layout"
	"yutug.lol/spark/internal/ui/components"
)

func (win *Window) layoutTabBar(gtx layout.Context) layout.Dimensions {
	// Sync TabBar.Tabs slice with the current tab list.
	for len(win.tabBar.Tabs) < len(win.tabs) {
		win.tabBar.Tabs = append(win.tabBar.Tabs, &components.TabState{})
	}
	win.tabBar.Tabs = win.tabBar.Tabs[:len(win.tabs)]

	// Point each TabState at the Tab's existing widget state.
	for i, tab := range win.tabs {
		win.tabBar.Tabs[i] = &tab.State
	}

	dims, result := win.tabBar.Layout(gtx, win.theme, win.activeTab)

	switch result.Event {
	case components.TabBarSwitchN:
		win.activeTab = result.SwitchedTo
		win.w.Invalidate()
	case components.TabBarCloseN:
		win.tabs[result.ClosedIdx].closed = true
		win.w.Invalidate()
	}

	return dims
}
