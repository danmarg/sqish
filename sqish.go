// Package sqish implements the basic functionality to store and retrieve shell histories.
package sqish

import (
	"database/sql"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

const sqlSchema = `
create table history (
  cmd text,
  dir text,
  hostname text,
  shell_pid integer,
  time integer
);`

// Record holds the data recorded for a single shell command.
type Record struct {
	Cmd      string
	Dir      string
	Hostname string
	ShellPid int
	Time     time.Time
}

// Query holds the components of a database query.
type Query struct {
	Q        string
	Dir      *string
	Hostname *string
	ShellPid int
	bool     SortByFreq
	Limit    int
}

// Database holds a database connection and provides insert and retrieval.
type Database interface {
	Add(Record) error
	Close() error
}

type sqlDatabase struct {
	db *sql.DB
	Database
}

func NewDatabase(path string) (Database, error) {
	d := &sqlDatabase{}
	// Check if the DB already exists, or if we must create the table.
	_, err := os.Stat(path)
	n := os.IsNotExist(err)
	d.db, err = sql.Open("sqlite3", path)
	// If new, we must create the table.
	if n {
		if _, err := d.db.Exec(sqlSchema); err != nil {
			return nil, err
		}
	}
	return d, err
}

func (d *sqlDatabase) Add(r Record) error {
	_, err := sq.Insert("history").
		Columns("cmd", "dir", "hostname", "shell_pid", "time").
		Values(r.Cmd, r.Dir, r.Hostname, r.ShellPid, r.Time.UnixNano()).
		RunWith(d.db).Exec()

	return err
}

func (d *sqlDatabase) Close() error {
	return d.db.Close()
}
