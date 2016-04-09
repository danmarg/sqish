package main

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode"
)

const (
	OutputBold   = "\033[1m"
	OutputNormal = "\033[0m"
	OutputBell   = "\a"
	OutputReset  = "\x1b[2K\r"

	InputBackspace = "\x7f"
	InputEnter     = "\n"
	InputUp        = "\027[A"
	InputDown      = "\027[B"
)

func readStdin(out chan string, kill chan bool) {
	//no buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	//no visible output
	exec.Command("stty", "-F", "/dev/tty", "-echo").Run()

	var b []byte = make([]byte, 2)
	for {
		select {
		case <-kill:
			return
		default:
			c, err := os.Stdin.Read(b)
			if err != io.EOF && err != nil {
				panic(err)
			}
			if c > 0 {
				out <- string(b[:c])
			}
		}
	}
}

func print(s string) {
	os.Stdout.Write([]byte(s))
}

func cliFindAsYouType(db database) error {
	var q, r string
	offset := 0
	defer func() {
		exec.Command("stty", "-f", "/dev/tty", "echo").Run()
		os.Stderr.WriteString(r)
	}()
	stdin := make(chan string, 1)
	kill := make(chan bool, 1)
	print("> ")
	go readStdin(stdin, kill)
	for {
		c := <-stdin
		if c == InputEnter {
			return nil
		} else if c == InputBackspace {
			if len(q) > 0 {
				q = q[:len(q)-1]
			} else {
				print(OutputBell)
			}
		} else if c == InputUp {
			offset += 1
		} else if c == InputDown {
			if offset > 0 {
				offset -= 1
			}
		} else {
			printable := true
			for _, char := range c {
				if !unicode.IsPrint(char) {
					printable = false
					break
				}
			}
			if printable {
				q += string(c)
			}
		}
		rs, e := db.Query(query{
			Cmd:    &q,
			Limit:  1,
			Offset: offset,
		})
		if e != nil {
			return e
		}
		if len(rs) == 0 {
			r = ""
			print(OutputReset + "> " + q + OutputBell)
		} else {
			r = rs[0].Cmd
			i := strings.Index(r, q)
			print(OutputReset +
				"> " +
				r[:i] +
				OutputBold +
				r[i:i+len(q)] +
				OutputNormal +
				r[i+len(q):])
		}
	}

}
