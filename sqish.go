// Package sqish implements the basic functionality to store and retrieve shell histories.
package sqish

import (
	"os"
	"time"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

// Record holds the data recorded for a single shell command.
type Record struct {
	Cmd            string
	Dir            string
	Hostname       string
	ShellSessionId string
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

func orEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (d *sqlDatabase) Query(q Query) ([]Record, error) {
	var rs []Record
	// TODO: filter on other variables.
	err := d.db.Where("cmd LIKE ?", "%"+orEmpty(q.Cmd)+"%").
		// TODO: Allow sort by frequency.
		Order("time").Find(&rs).Error
	return rs, err
}
