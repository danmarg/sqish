package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

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
	// GUI state.
	gui            *gocui.Gui
	db             database
	shellSessionID string
	queries        chan query
	results        chan []record
	resultsOffset  int
	once           sync.Once
	// Settings.
	set setting
	// Currently-displayed results.
	currentResults []record
	// Key binding map.
	keybindings = map[gocui.Key]gocui.KeybindingHandler{
		gocui.KeyCtrlC: quit,
		gocui.KeyCtrlD: quit,
		gocui.KeyCtrlS: func(g *gocui.Gui, v *gocui.View) error {
			set.SortByFreq = !set.SortByFreq
			return drawSettings(v)
		},
		gocui.KeyCtrlL: func(g *gocui.Gui, v *gocui.View) error {
			set.OnlyMySession = !set.OnlyMySession
			return drawSettings(v)
		},
		gocui.KeyCtrlW: func(g *gocui.Gui, v *gocui.View) error {
			set.OnlyMyCwd = !set.OnlyMyCwd
			return drawSettings(v)
		},
		gocui.KeyArrowUp:   func(g *gocui.Gui, v *gocui.View) error { return moveResultLine(true) },
		gocui.KeyArrowDown: func(g *gocui.Gui, v *gocui.View) error { return moveResultLine(false) },
		gocui.KeyEnter: func(g *gocui.Gui, v *gocui.View) error {
			v, err := g.View(resultsWindow)
			if err != nil {
				return err
			}
			if resultsOffset < len(currentResults) {
				os.Stderr.WriteString(currentResults[resultsOffset].Cmd)
			}
			return quit(g, v)
		},
	}
)

func runGui(d database, shellID, initialQuery string) error {
	var err error
	gui = gocui.NewGui()
	db = d
	set, err = db.Setting()
	if err != nil {
		return err
	}
	shellSessionID = shellID
	if err := gui.Init(); err != nil {
		return err
	}
	defer gui.Close()
	gui.SetLayout(func(g *gocui.Gui) error {
		if err := layout(g); err != nil {
			return err
		}
		// Prefill the search bar. This is ugly, I admit.
		once.Do(func() {
			v, err := g.View(searchBar)
			if err != nil {
				panic(err)
			}
			for _, c := range initialQuery {
				v.EditWrite(c)
			}
			if err := findAsYouType(shellSessionID, db, queries); err != nil {
				panic(err)
			}
		})
		return nil
	})
	if err := setKeybindings(); err != nil {
		return err
	}
	// Channels for sending queries and getting responses.
	queries = make(chan query, bufSize)
	results = make(chan []record, bufSize)
	// Set the editor to do find-as-you-type.
	gui.Editor = gocui.EditorFunc(func(v *gocui.View, k gocui.Key, c rune, m gocui.Modifier) {
		if _, ok := keybindings[k]; ok {
			return
		}
		gocui.DefaultEditor.Edit(v, k, c, m)
		findAsYouType(shellSessionID, db, queries)
	})
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
	err = gui.MainLoop()
	if err != nil && err != gocui.ErrQuit {
		return err
	}
	return nil
}

func quit(_ *gocui.Gui, _ *gocui.View) error {
	if err := db.WriteSetting(&set); err != nil {
		return err
	}
	return gocui.ErrQuit
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
		v.Wrap = true
		v.Highlight = true
		v.SelBgColor, v.SelFgColor = v.FgColor, v.BgColor
	}
	// searchTitle
	searchTxt := "Search:"
	if v, err := g.SetView(searchTitle, -1, maxY-4, len(searchTxt), maxY-2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		v.FgColor = gocui.AttrBold
		fmt.Fprint(v, searchTxt)
	}
	// searchBar
	if v, err := g.SetView(searchBar, len(searchTxt), maxY-4, maxX, maxY-2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
		v.Wrap = true
		v.Frame = false
	}
	// toolBar
	if v, err := g.SetView(toolBar, -1, maxY-2, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Frame = false
		drawSettings(v)
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

func findAsYouType(shellSessionID string, db database, qs chan<- query) error {
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
	q := query{
		Cmd:        &s,
		SortByFreq: set.SortByFreq,
	}
	if set.OnlyMySession {
		q.Hostname = &h
		q.ShellSessionID = &shellSessionID
	}
	if set.OnlyMyCwd {
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
	if up && resultsOffset > 0 {
		v.MoveCursor(0, -1, false)
		resultsOffset--
	} else if !up && resultsOffset < len(currentResults)-1 {
		v.MoveCursor(0, 1, false)
		resultsOffset++
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
	var a, b, s, c, d rune
	if set.SortByFreq {
		a, b, s = ' ', '*', 'f'
	} else {
		a, b, s = '*', ' ', 't'
	}
	// leftL is the "long" option; leftS is the "short" one.
	leftL := fmt.Sprintf("[^S]ort by [%c] time [%c] freq", a, b)
	leftS := fmt.Sprintf("[^S][%c]", s)
	// Middle (long and short).
	if set.OnlyMySession {
		c = '*'
	} else {
		c = ' '
	}
	middleL := fmt.Sprintf("[^L]imit to my session [%c]", c)
	middleS := fmt.Sprintf("[^L][%c]", c)

	// Right (long and short).
	if set.OnlyMyCwd {
		d = '*'
	} else {
		d = ' '
	}
	rightL := fmt.Sprintf("[^W]orking dir only [%c]", d)
	rightS := fmt.Sprintf("[^W][%c]", d)

	// Now choose the long or short form and pad it.
	var left, middle, right string
	if len(leftL)+len(middleL)+len(rightL) < maxX {
		left, middle, right = leftL, middleL, rightL
	} else {
		left, middle, right = leftS, middleS, rightS
	}
	var lpad, rpad []byte
	if len(left)+len(middle)/2 >= maxX/2 {
		lpad = []byte{}
	} else {
		lpad = make([]byte, maxX/2-len(left)-len(middle)/2)
	}
	if len(right)+len(middle)/2 >= maxX/2 {
		rpad = []byte{}
	} else {
		rpad = make([]byte, maxX/2-len(right)-len(middle)/2)
	}
	fmt.Fprint(v, left+string(lpad)+middle+string(rpad)+right)
	findAsYouType(shellSessionID, db, queries)
	return nil
}

func drawResults(rs []record) error {
	v, err := gui.View(resultsWindow)
	if err != nil {
		return err
	}
	v.Clear()
	v.SetCursor(0, 0)
	resultsOffset = 0
	currentResults = rs
	for _, r := range rs {
		fmt.Fprintf(v, "%s\t\t|\t\t%s\n", r.Time.Format("2006/01/02 15:04:05"), strings.Replace(r.Cmd, "\n", " ", -1))
	}
	return nil
}
