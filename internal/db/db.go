package db

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func New(addr string, maxOpenConns, maxIdleCons int, maxIdleTime string) (*sql.DB, error) {
	log.Println(addr)
	db, err := sql.Open("postgres", addr)
	if err != nil {
		return nil, err
	}

	maxIdleTimeParse, err := time.ParseDuration(maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxIdleTime(maxIdleTimeParse)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleCons)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}
