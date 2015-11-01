// Package sqish implements the basic functionality to store and retrieve shell histories.
package sqish

import (
	"os"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

const (
	sqlSelectByFreq = "cmd, dir, '' as hostname, '' as shell_session_id, time" // TODO: do max(time) here.
	sqlGroupByFreq  = "cmd, dir"
	sqlSelectByDate = "cmd, dir, hostname, shell_session_id, time"
)

// Record holds the data recorded for a single shell command.
type Record struct {
	Cmd            string `sql:"size:65535"`
	Dir            string `sql:"size:65535"`
	Hostname       string `sql:"size:65535"`
	ShellSessionId string `sql:"size:65535"`
	Time           time.Time
}

// Query holds the components of a database query.
type Query struct {
	Cmd            *string
	Dir            *string
	Hostname       *string
	ShellSessionId *string
	SortByFreq     bool
	Limit          int
}

// Database holds a database connection and provides insert and retrieval.
type Database interface {
	Add(*Record) error
	Close() error
	Query(q Query) ([]Record, error)
}

type sqlDatabase struct {
	db gorm.DB
	Database
}

func NewDatabase(path string) (Database, error) {
	d := &sqlDatabase{}
	// Check if the DB already exists, or if we must create the table.
	_, err := os.Stat(path)
	n := os.IsNotExist(err)
	d.db, err = gorm.Open("sqlite3", path)
	// If new, we must create the table.
	if n {
		err = d.db.CreateTable(&Record{}).Error
	}
	return d, err
}

func (d *sqlDatabase) Add(r *Record) error {
	return d.db.Create(r).Error
}

func (d *sqlDatabase) Close() error {
	return d.db.Close()
}

func (d *sqlDatabase) Query(q Query) ([]Record, error) {
	var rs []Record
	var ws []string
	var ps []interface{}
	if q.Cmd != nil {
		ws = append(ws, "cmd LIKE ?")
		ps = append(ps, "%"+*q.Cmd+"%")
	}
	if q.Dir != nil {
		ws = append(ws, "dir = ?")
		ps = append(ps, q.Dir)
	}
	if q.Hostname != nil {
		ws = append(ws, "hostname = ?")
		ps = append(ps, q.Hostname)
	}
	if q.ShellSessionId != nil {
		ws = append(ws, "shell_session_id = ?")
		ps = append(ps, q.ShellSessionId)
	}
	var db *gorm.DB
	if q.SortByFreq {
		db = d.db.Table("records").Select(sqlSelectByFreq).Where(strings.Join(ws, " "), ps...).Group(sqlGroupByFreq)
	} else {
		db = d.db.Table("records").Select(sqlSelectByDate).Where(strings.Join(ws, " "), ps...)
	}
	err := db.Scan(&rs).Error
	return rs, err
}
