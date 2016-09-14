package crawler

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/lib/pq"
)

type Storage interface {
	Load() (*State, error)
	SetStatus(url string, t ItemType, s Status) error
	Close() error
}

var _ Storage = new(PGStorage)

// PGStorage write statuses to postgres
type PGStorage struct {
	db        *sql.DB
	tableName string
}

// NewPGStorage return new postgres storage instance
func NewPGStorage(connection string, mainHost string) (*PGStorage, error) {
	db, err := sql.Open("postgres", connection)
	if err != nil {
		return nil, err
	}
	return &PGStorage{
		db:        db,
		tableName: mainHost,
	}, nil
}

// Close db connections
func (pgs *PGStorage) Close() error {
	return pgs.db.Close()
}

// Load state from db
func (pgs *PGStorage) Load() (*State, error) {
	state := NewState(pgs)

	rws, err := pgs.db.Query(pgs.getQuery(queryGetAll))
	if err != nil {
		log.Printf("todo need catch table not exists error, gotten: %s", err)
		return nil, err
	}

	var i Item
	var url string
	for rws.Next() {
		err := rws.Scan(&url, &i.itype, &i.status)
		if err != nil {
			return nil, err
		}
		state.AddProgress(url, i)
	}
	state.SetEmpty(false)
	return state, nil
}

// Clear truncate table adn return empty state
func (pgs *PGStorage) Clear() (*State, error) {
	_, err := pgs.db.Query(pgs.getQuery(queryClearTable))
	if err != nil {
		if pgerr, ok := err.(*pq.Error); ok {
			if pgerr.Code == "42P01" {
				_, err = pgs.db.Query(pgs.getQuery(queryCreateTableTpl))
				if err != nil {
					log.Printf("create table error: %s", err)
				}
			}
		} else {
			log.Printf("truncate table error: %#v", err)
		}
	}
	return NewState(pgs), nil
}

// SetStatus save status for concret url
func (pgs *PGStorage) SetStatus(url string, t ItemType, s Status) error {
	_, err := pgs.db.Exec(pgs.getQuery(querySetStatus), url, t, s)
	if err != nil {
		log.Printf("set status error: %#v", err)
	}
	return err
}

func (pgs *PGStorage) getQuery(tpl string) string {
	return fmt.Sprintf(tpl, pgs.tableName)
}
