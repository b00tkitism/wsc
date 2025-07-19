package main

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"time"
)

type Database struct {
	File   string
	Handle *sql.DB
}

func (db *Database) Start(ctx context.Context) error {
	dbHandle, err := sql.Open("sqlite3", db.File)
	if err != nil {
		return err
	}
	if err := dbHandle.PingContext(ctx); err != nil {
		dbHandle.Close()
		return err
	}
	db.Handle = dbHandle
	if err := db.createTables(ctx); err != nil {
		return err
	}
	return nil
}

func (db *Database) Stop() error {
	return db.Handle.Close()
}

func (db *Database) UpdateUser(ctx context.Context, id int64, usedTraffic int64) error {
	statement, err := db.Handle.PrepareContext(ctx, "UPDATE `users` SET `used_traffic`=`used_traffic`+? WHERE `id`=?")
	if err != nil {
		return err
	}
	defer statement.Close()
	if _, err := statement.ExecContext(ctx, usedTraffic, id); err != nil {
		return err
	}

	return nil
}

func (db *Database) ChargeUserService(ctx context.Context, auth string, rate int64, totalTraffic int64, duration time.Duration) error {
	now := time.Now().UnixNano()
	endTime := now + int64(duration)
	statement, err := db.Handle.PrepareContext(ctx, "UPDATE `users` SET `rate`=?, `used_traffic`='0', `total_traffic`=?, `start_time`=?, `end_time`=? WHERE `auth`=?")
	if err != nil {
		return err
	}
	defer statement.Close()
	if _, err := statement.ExecContext(ctx, rate, totalTraffic, now, endTime, auth); err != nil {
		return err
	}
	return nil
}

func (db *Database) FindUser(ctx context.Context, auth string) (id int64, rate int64, usedTraffic int64, totalTraffic int64, endTime int64, err error) {
	query, err := db.Handle.QueryContext(ctx, "SELECT `id`, `rate`, `used_traffic`, `total_traffic`, `end_time` FROM `users` WHERE `auth`=?", auth)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	defer query.Close()

	query.Next()
	if err := query.Scan(&id, &rate, &usedTraffic, &totalTraffic, &endTime); err != nil {
		return 0, 0, 0, 0, 0, err
	}

	return
}

func (db *Database) CreateUser(ctx context.Context, auth string, rate int64, totalTraffic int64, duration time.Duration) error {
	now := time.Now().UnixNano()
	end := now + int64(duration)
	statement, err := db.Handle.PrepareContext(ctx, `INSERT INTO users('auth', 'rate', 'total_traffic', 'start_time', 'end_time') SELECT ?, ?, ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM users WHERE auth=?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.ExecContext(ctx, auth, rate, totalTraffic, now, end, auth)
	if err != nil {
		return err
	}
	return nil
}

func (db *Database) createTables(ctx context.Context) error {
	statement, err := db.Handle.PrepareContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
		    'id'            INTEGER NOT NULL UNIQUE,
		    'auth'          TEXT NOT NULL,
		    'rate'          INTEGER NOT NULL DEFAULT '10485760',
		    'used_traffic'  INTEGER NOT NULL DEFAULT '0',
		    'total_traffic' INTEGER NOT NULL DEFAULT '0',
		    'start_time'    INTEGER NOT NULL DEFAULT '0',
		    'end_time'      INTEGER NOT NULL DEFAULT '0',
		    PRIMARY KEY('id' AUTOINCREMENT)
		)
	`)
	if err != nil {
		return err
	}
	defer statement.Close()
	_, err = statement.ExecContext(ctx)
	return err
}
