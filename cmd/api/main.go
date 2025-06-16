package main

import (
	"log"

	"github.com/MisterDodik/Barbershop/internal/db"
	"github.com/MisterDodik/Barbershop/internal/env"
	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/joho/godotenv"
)

const version = "0.0.1"

func main() {
	godotenv.Load()
	cfg := config{
		addr: env.GetString("ADDR", ":3000"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://postgres:admin@localhost/barbershop?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_IDLE_OPEN_CONNS", 30),
			maxIdleTime:  env.GetString("DB_IDLE_TIME", "15m"),
		},
	}

	db, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		log.Panic(err)
	}
	store := store.NewStorage(db)

	app := &application{
		config: cfg,
		store:  store,
	}

	mux := app.mount()
	if err := app.run(mux); err != nil {
		log.Fatal(err)
	}
}
