package main

import (
	"fmt"

	"github.com/danmarg/sqish"
	"github.com/jroimartin/gocui"
)

const (
	bufSize = 128 // How many queries or results to queue during async find-as-you-type.
	// View names.
	resultsWindow = "resultsWindow"
	searchTitle   = "searchTitle"
	searchBar     = "searchBar"
	toolBar       = "toolBar"
)

var (
	keybindings = map[gocui.Key]gocui.KeybindingHandler{
		gocui.KeyCtrlC: func(_ *gocui.Gui, _ *gocui.View) error { return gocui.Quit },
		gocui.KeyCtrlD: func(_ *gocui.Gui, _ *gocui.View) error { return gocui.Quit },
		gocui.KeyCtrlS: func(g *gocui.Gui, v *gocui.View) error { sortByFreq = !sortByFreq; return drawSettings(g, v) },
		gocui.KeyCtrlL: func(g *gocui.Gui, v *gocui.View) error { onlyMySession = !onlyMySession; return drawSettings(g, v) },
		gocui.KeyCtrlW: func(g *gocui.Gui, v *gocui.View) error { onlyMyCwd = !onlyMyCwd; return drawSettings(g, v) },
	}
	// State.
	sortByFreq    = false // True = sort by frequency; false = sort by time.
	onlyMySession = false // True = limit to results in current shell; false = no limit.
	onlyMyCwd     = false // True = limit to commands in the current dir; false = no limit.
)

func runGui(db sqish.Database) error {
	g := gocui.NewGui()
	if err := g.Init(); err != nil {
		return err
	}
	defer g.Close()
	g.SetLayout(layout)
	if err := setKeybindings(g); err != nil {
		return err
	}
	err := g.MainLoop()
	if err != nil && err != gocui.Quit {
		return err
	}
	// Set the editor to do find-as-you-type.
	//queries := make(chan sqish.Query, bufSize)
	//results := make(chan sqish.Record, bufSize)
	gocui.Edit = func(v *gocui.View, k gocui.Key, c rune, m gocui.Modifier) {
		findAsYouType(db, g)
		gocui.DefaultEditor(v, k, c, m)
	}
	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	/*
		Layout looks something like this:
		+--------------------------------------------------+
		|            resultsWindow                         |
		|                                                  |
		|+------------------------+------------------------+
		||  searchTitle           | searchBar             ||
		|+------------------------+------------------------|
		||  toolBar                                       ||
		+-+-----------------------------------------------++
	*/
	// resultsWindow
	if v, err := g.SetView(resultsWindow, -1, -1, maxX, maxY-4); err != nil {
		v.Frame = true
		v.Highlight = false
	}
	// searchTitle
	searchTxt := "Search:"
	if v, err := g.SetView(searchTitle, -1, maxY-4, len(searchTxt), maxY-2); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		v.Frame = false
		v.FgColor = gocui.AttrBold
		fmt.Fprint(v, searchTxt)
	}
	// searchBar
	if v, err := g.SetView(searchBar, len(searchTxt), maxY-4, maxX, maxY-2); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		v.Editable = true
		v.Frame = false
	}
	// toolBar
	if v, err := g.SetView(toolBar, -1, maxY-2, maxX, maxY); err != nil {
		if err != gocui.ErrorUnkView {
			return err
		}
		v.Frame = false
		drawSettings(g, v)
		left := "[^S]ort by [*] time [ ] freq"
		right := "[^L]imit to my session [ ]"
		middle := make([]byte, maxX-len(left)-len(right))
		fmt.Fprint(v, left+string(middle)+right)
	}

	return g.SetCurrentView(searchBar)
}

func setKeybindings(g *gocui.Gui) error {
	for k, v := range keybindings {
		if err := g.SetKeybinding("", k, gocui.ModNone, v); err != nil {
			return err
		}
	}
	return nil
}

func findAsYouType(db sqish.Database, g *gocui.Gui) error {
	v, err := g.View(searchBar)
	if err != nil {
		return err
	}
	_ = v.Buffer()

	return nil
}

func drawSettings(g *gocui.Gui, v *gocui.View) error {
	v, err := g.View(toolBar)
	if err != nil {
		return err
	}
	v.Clear()
	maxX, _ := g.Size()
	var a, b, c, d rune
	if sortByFreq {
		a, b = ' ', '*'
	} else {
		a, b = '*', ' '
	}
	left := fmt.Sprintf("[^S]ort by [%c] time [%c] freq", a, b)
	if onlyMySession {
		c = '*'
	} else {
		c = ' '
	}
	middle := fmt.Sprintf("[^L]imit to my session [%c]", c)
	if onlyMyCwd {
		d = '*'
	} else {
		d = ' '
	}
	right := fmt.Sprintf("[^W]orking dir only [%c]", d)
	lpad := make([]byte, maxX/2-len(left)-len(middle)/2)
	rpad := make([]byte, maxX/2-len(right)-len(middle)/2)
	fmt.Fprint(v, left+string(lpad)+middle+string(rpad)+right)
	return nil
}
