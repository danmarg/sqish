package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/danmarg/sqish"
)

func main() {
	app := cli.NewApp()
	app.Name = "SQISH"
	app.Usage = "SQL Interactive Shell History"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "database, d",
			Value: os.ExpandEnv("${HOME}/.sqish_db"),
			Usage: "Path to database",
		},
		cli.StringFlag{
			Name:  "shell_session_id",
			Value: os.ExpandEnv("$SQISH_SESSION_ID"),
			Usage: "Shell session ID. This is used to uniquely identify a shell session.",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "Add a command to the history.",
			Action: func(ctx *cli.Context) {
				runWithErr(func() error {
					db, err := sqish.NewDatabase(ctx.GlobalString("database"))
					if err != nil {
						return err
					}
					defer db.Close()
					// Fill a record.
					wd, err := os.Getwd()
					if err != nil {
						return err
					}
					h, err := os.Hostname()
					if err != nil {
						return err
					}
					r := sqish.Record{
						Cmd:            strings.Join(ctx.Args(), " "),
						Dir:            wd,
						Hostname:       h,
						ShellSessionId: ctx.GlobalString("shell_session_id"),
						Time:           time.Now(),
					}
					return db.Add(r)
				})
			},
		},
		{
			Name:    "search",
			Aliases: []string{"s"},
			Usage:   "Search backwards",
			Action: func(ctx *cli.Context) {
				runWithErr(
					func() error {
						db, err := sqish.NewDatabase(ctx.GlobalString("database"))
						if err != nil {
							return err
						}
						defer db.Close()
						return runGui(db, ctx.GlobalString("shell_session_id"))
					})

			},
		},
	}
	app.Run(os.Args)
}

func runWithErr(fn func() error) {
	if err := fn(); err != nil {
		fmt.Println("Error: ", err)
	}
}
