package database

import (
	"database/sql"
	"fmt"
)

type Tx struct {
	*sql.Tx
	Driver string
}

func (tx *Tx) execer() Execer {
	return tx.Tx
}

func (tx *Tx) driver() string {
	return tx.Driver
}

func (db *DB) Transaction(fn func(tx *Tx) error) error {
	sqlTx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("[error] Failed to start transaction: %w", err)
	}

	tx := &Tx{
		Tx:     sqlTx,
		Driver: db.Driver,
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("[error] Transaction failed: %v (rollback error: %w)", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("[error] Failed to commit transaction: %w", err)
	}

	return nil
}

func (tx *Tx) SetTable(model interface{}) error {
	return setTable(tx, model)
}

func (db *DB) AutoMigrate(models ...interface{}) error {
	if db.Driver == "postgres" {
		return db.Transaction(func(tx *Tx) error {
			for _, model := range models {
				if err := tx.SetTable(model); err != nil {
					return err
				}
			}
			return nil
		})
	}

	for _, model := range models {
		if err := db.SetTable(model); err != nil {
			return err
		}
	}

	return nil
}