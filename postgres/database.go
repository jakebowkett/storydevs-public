package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

const errSerial = "pq: could not serialize access due to read/write dependencies among transactions"

/*
	This error will name the conflicting constraint so
	we only have the start of the error here and check
	for it as a prefix below.
*/
const errUnique = "pq: duplicate key value violates unique constraint"

func Retry(err error) bool {
	e := err.Error()
	if e == errSerial {
		return true
	}
	if strings.HasPrefix(e, errUnique) {
		return true
	}
	return false
}

func withQuery(err error, query string) error {
	if err == nil {
		return err
	}
	if err.Error() == errSerial {
		return err
	}
	return fmt.Errorf("%w:\n%s", err, query)
}

type DB struct {
	*sqlx.DB
	db *sqlx.DB
}

func MustConnect(connStr string) sd.DB {
	db, err := Connect(connStr)
	if err != nil {
		panic(err)
	}
	return db
}

func Connect(connStr string) (sd.DB, error) {
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return &DB{db, db}, nil
}

func (db *DB) Get(dest interface{}, query string, args ...interface{}) (err error) {
	err = db.db.Get(dest, query, args...)
	err = withQuery(err, query)
	return err
}

func (db *DB) Select(dest interface{}, query string, args ...interface{}) (err error) {
	err = db.db.Select(dest, query, args...)
	err = withQuery(err, query)
	return err
}

func (db *DB) Query(query string, args ...interface{}) (rows sd.DbRows, err error) {
	rows, err = db.db.Queryx(query, args...)
	err = withQuery(err, query)
	return rows, err
}

func (db *DB) Exists(fromWhere string, args ...interface{}) (exists bool, err error) {
	err = db.Get(&exists, `SELECT EXISTS (SELECT null `+fromWhere+`)`, args...)
	err = withQuery(err, fromWhere)
	return exists, err
}

func (db *DB) Exec(query string, args ...interface{}) (result sd.DbResult, err error) {
	result, err = db.db.Exec(query, args...)
	err = withQuery(err, query)
	return result, err
}

type Tx struct {
	*sqlx.Tx
	tx *sqlx.Tx
}

func (db *DB) Begin() (sd.Tx, error) {
	return db.begin(false)
}

func (db *DB) BeginRead() (sd.Tx, error) {
	return db.begin(true)
}

func (db *DB) begin(readOnly bool) (sd.Tx, error) {
	tx, err := db.db.BeginTxx(
		context.TODO(),
		&sql.TxOptions{
			Isolation: sql.LevelSerializable,
			ReadOnly:  readOnly,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Tx{tx, tx}, nil
}

func (tx *Tx) Rollback(err error) error {
	txErr := tx.tx.Rollback()
	if err == nil {
		return txErr
	}
	if txErr == nil {
		return err
	}
	return errors.New(err.Error() + ": " + txErr.Error())
}

func (tx *Tx) Get(dest interface{}, query string, args ...interface{}) (err error) {
	err = tx.tx.Get(dest, query, args...)
	err = withQuery(err, query)
	return err
}

func (tx *Tx) Select(dest interface{}, query string, args ...interface{}) (err error) {
	err = tx.tx.Select(dest, query, args...)
	err = withQuery(err, query)
	return err
}
func (tx *Tx) Query(query string, args ...interface{}) (rows sd.DbRows, err error) {
	rows, err = tx.tx.Queryx(query, args...)
	err = withQuery(err, query)
	return rows, err
}

func (tx *Tx) Exists(fromWhere string, args ...interface{}) (exists bool, err error) {
	err = tx.tx.Get(&exists, `SELECT EXISTS (SELECT null `+fromWhere+`)`, args...)
	err = withQuery(err, fromWhere)
	return exists, err
}

func (tx *Tx) Exec(query string, args ...interface{}) (result sd.DbResult, err error) {
	result, err = tx.tx.Exec(query, args...)
	err = withQuery(err, query)
	return result, err
}
