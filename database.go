package main

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

// setting holds the persisted settings.
type setting struct {
	ID            int
	SortByFreq    bool
	OnlyMySession bool
	OnlyMyCwd     bool
}

// record holds the data recorded for a single shell command.
type record struct {
	Cmd            string `sql:"size:65535"`
	Dir            string `sql:"size:65535"`
	Hostname       string `sql:"size:65535"`
	ShellSessionId string `sql:"size:65535"`
	Time           time.Time
}

// query holds the components of a database query.
type query struct {
	Cmd            *string
	Dir            *string
	Hostname       *string
	ShellSessionId *string
	SortByFreq     bool
	Limit          int
}

// database holds a database connection and provides insert and retrieval.
type database interface {
	Add(*record) error
	Close() error
	Query(query) ([]record, error)
	Setting() (setting, error)
	WriteSetting(*setting) error
}

type sqlDatabase struct {
	db gorm.DB
	database
}

func newDatabase(path string) (database, error) {
	d := &sqlDatabase{}
	// Check if the DB already exists, or if we must create the table.
	_, err := os.Stat(path)
	n := os.IsNotExist(err)
	d.db, err = gorm.Open("sqlite3", path)
	// If new, we must create the table.
	if n {
		if err := d.db.CreateTable(&setting{}).Error; err != nil {
			return nil, err
		}
		if err := d.db.CreateTable(&record{}).Error; err != nil {
			return nil, err
		}
	}
	return d, err
}

func (d *sqlDatabase) Add(r *record) error {
	return d.db.Create(r).Error
}

func (d *sqlDatabase) Close() error {
	return d.db.Close()
}

func (d *sqlDatabase) Query(q query) ([]record, error) {
	var rs []record
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
		db = d.db.Table("records").Select(sqlSelectByFreq).Where(strings.Join(ws, " and "), ps...).Group(sqlGroupByFreq)
	} else {
		db = d.db.Table("records").Select(sqlSelectByDate).Where(strings.Join(ws, " and "), ps...)
	}
	err := db.Scan(&rs).Error
	return rs, err
}

func (d *sqlDatabase) Setting() (setting, error) {
	var s setting
	r := d.db.First(&s)
	if r.RecordNotFound() {
		return s, nil
	}
	return s, r.Error
}

func (d *sqlDatabase) WriteSetting(s *setting) error {
	return d.db.Save(s).Error
}
