package crawler

import "database/sql"

type Storage interface {
	Load() (*State, error)
	SetStatus(url string, t ItemType, s Status) error
}

// PGStorage write statuses to postgres
type PGStorage struct {
	db        *sql.DB
	tableName string
}

// NewPGStorage return new postgres storage instance
func NewPGStorage(connection, mainHost string) (*PGStorage, error) {
	return &PGStorage{}, nil
}

// Load state from db
func (pgs *PGStorage) Load() (*State, error) {
	return NewState(pgs), nil
}

// Clear truncate table adn return empty state
func (pgs *PGStorage) Clear() (*State, error) {
	return nil, nil
}

// SetStatus save status for concret url
func (pgs *PGStorage) SetStatus(url string, t ItemType, s Status) error {
	return nil
}
