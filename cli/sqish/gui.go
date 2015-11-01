package main

import (
	"fmt"
	"os"
	"strings"

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
		gocui.KeyCtrlC:     func(_ *gocui.Gui, _ *gocui.View) error { return gocui.Quit },
		gocui.KeyCtrlD:     func(_ *gocui.Gui, _ *gocui.View) error { return gocui.Quit },
		gocui.KeyCtrlS:     func(g *gocui.Gui, v *gocui.View) error { sortByFreq = !sortByFreq; return drawSettings(v) },
		gocui.KeyCtrlL:     func(g *gocui.Gui, v *gocui.View) error { onlyMySession = !onlyMySession; return drawSettings(v) },
		gocui.KeyCtrlW:     func(g *gocui.Gui, v *gocui.View) error { onlyMyCwd = !onlyMyCwd; return drawSettings(v) },
		gocui.KeyArrowUp:   func(g *gocui.Gui, v *gocui.View) error { return moveResultLine(true) },
		gocui.KeyArrowDown: func(g *gocui.Gui, v *gocui.View) error { return moveResultLine(false) },
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			v, err := g.View(resultsWindow)
			if err != nil {
				return err
			}
			x, _ := v.Cursor()
			if x < len(currentResults) {
				os.Stderr.WriteString(currentResults[x].Cmd)
			}
			return gocui.Quit
		},
	}
	// GUI state.
	gui *gocui.Gui
	// Settings.
	sortByFreq    = false // True = sort by frequency; false = sort by time.
	onlyMySession = false // True = limit to results in current shell; false = no limit.
	onlyMyCwd     = false // True = limit to commands in the current dir; false = no limit.
	// Currently-displayed results.
	currentResults []sqish.Record
)

func runGui(db sqish.Database, shellSessionId string) error {
	gui = gocui.NewGui()
	if err := gui.Init(); err != nil {
		return err
	}
	defer gui.Close()
	gui.SetLayout(layout)
	if err := setKeybindings(); err != nil {
		return err
	}
	// Channels for sending queries and getting responses.
	queries := make(chan sqish.Query, bufSize)
	results := make(chan []sqish.Record, bufSize)
	// Set the editor to do find-as-you-type.
	gocui.Edit = func(v *gocui.View, k gocui.Key, c rune, m gocui.Modifier) {
		if _, ok := keybindings[k]; ok {
			return
		}
		gocui.DefaultEditor(v, k, c, m)
		findAsYouType(shellSessionId, db, queries)
	}
	// Async function to execute queries.
	go func() {
		for q := range queries {
			rs, err := db.Query(q)
			if err != nil {
				panic(err)
			}
			results <- rs
		}
	}()
	// Async function to draw results.
	go func() {
		for rs := range results {
			if err := drawResults(rs); err != nil {
				panic(err)
			}
		}
	}()
	// Start GUI loop.
	err := gui.MainLoop()
	if err != nil && err != gocui.Quit {
		return err
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
		v.Highlight = true
		v.SelBgColor, v.SelFgColor = v.FgColor, v.BgColor
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
		drawSettings(v)
		left := "[^S]ort by [*] time [ ] freq"
		right := "[^L]imit to my session [ ]"
		middle := make([]byte, maxX-len(left)-len(right))
		fmt.Fprint(v, left+string(middle)+right)
	}

	return g.SetCurrentView(searchBar)
}

func setKeybindings() error {
	for k, v := range keybindings {
		if err := gui.SetKeybinding("", k, gocui.ModNone, v); err != nil {
			return err
		}
	}
	return nil
}

func findAsYouType(shellSessionId string, db sqish.Database, qs chan<- sqish.Query) error {
	v, err := gui.View(searchBar)
	if err != nil {
		return err
	}
	s := strings.TrimSuffix(v.Buffer(), "\n")
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	h, err := os.Hostname()
	if err != nil {
		return err
	}
	q := sqish.Query{
		Cmd:        &s,
		SortByFreq: sortByFreq,
	}
	if onlyMySession {
		q.Hostname = &h
		q.ShellSessionId = &shellSessionId
	}
	if onlyMyCwd {
		q.Dir = &wd
	}
	qs <- q

	return nil
}

func moveResultLine(up bool) error {
	v, err := gui.View(resultsWindow)
	if err != nil {
		return err
	}
	if up {
		v.MoveCursor(0, -1, false)
	} else {
		if _, y := v.Cursor(); y+1 >= len(currentResults) {
			return nil
		}
		v.MoveCursor(0, 1, false)
	}
	return nil
}

func drawSettings(v *gocui.View) error {
	v, err := gui.View(toolBar)
	if err != nil {
		return err
	}
	v.Clear()
	maxX, _ := gui.Size()
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

func drawResults(rs []sqish.Record) error {
	_, maxY := gui.Size()
	v, err := gui.View(resultsWindow)
	if err != nil {
		return err
	}
	v.Clear()
	currentResults = rs
	for i := 0; i < len(rs) && i < maxY-4; i++ {
		fmt.Fprintf(v, "%s\t\t|\t\t%s\n", rs[i].Time.Format("2006/01/02 15:04:05"), rs[i].Cmd)
	}
	v.SetCursor(0, 0)
	gui.Flush()
	return nil
}
