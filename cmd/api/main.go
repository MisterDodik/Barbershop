package main

import (
	"log"
	"time"

	"github.com/MisterDodik/Barbershop/internal/auth"
	"github.com/MisterDodik/Barbershop/internal/db"
	"github.com/MisterDodik/Barbershop/internal/env"
	"github.com/MisterDodik/Barbershop/internal/mailer"
	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/joho/godotenv"
)

const version = "0.0.1"

func main() {
	godotenv.Load()
	cfg := config{
		BarbershopName: env.GetString("NAME", "Ime_Frizerskog_Salona"),
		addr:           env.GetString("ADDR", ":8080"),
		frontEndURL:    env.GetString("FRONTEND_URL", "localhost:3000"),
		env:            env.GetString("ENV", "development"),
		db: dbConfig{
			addr:         env.GetString("DB_ADDR", "postgres://postgres:admin@localhost/barbershop?sslmode=disable"),
			maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
			maxIdleConns: env.GetInt("DB_IDLE_OPEN_CONNS", 30),
			maxIdleTime:  env.GetString("DB_IDLE_TIME", "15m"),
		},
		auth: authConfig{
			basic: basicConfig{
				username: env.GetString("BASIC_AUTH_USERNAME", "admin"),
				password: env.GetString("BASIC_AUTH_PASSWORD", "admin"),
			},
			token: tokenConfig{
				secret:  env.GetString("AUTH_TOKEN_SECRET", "example"),
				expDate: time.Hour * 24 * 3,
				iss:     env.GetString("AUTH_TOKEN_ISSUER", "admin"),
			},
		},
		mail: mailConfig{
			fromEmail: env.GetString("FROM_EMAIL", "test@example.com"),
			exp:       time.Minute * 15,
			mailTrap: mailTrapConfig{
				apiKey:   env.GetString("MAILTRAP_API_KEY", ""),
				host:     env.GetString("MAILTRAP_HOST", ""),
				port:     env.GetInt("MAILTRAP_PORT", 587),
				username: env.GetString("MAILTRAP_USERNAME", ""),
				password: env.GetString("MAILTRAP_PASSWORD", ""),
			},
		},
	}

	db, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	if err != nil {
		log.Panic(err)
	}
	store := store.NewStorage(db)

	jwtAuthenticator := auth.NewJWTAuthenticator(cfg.auth.token.secret, cfg.auth.token.iss, cfg.auth.token.iss)

	mailer, err := mailer.NewMailTrapMailer(
		cfg.mail.mailTrap.apiKey,
		cfg.mail.fromEmail,
		cfg.mail.mailTrap.host,
		cfg.mail.mailTrap.username,
		cfg.mail.mailTrap.password,
		cfg.mail.mailTrap.port,
	)
	if err != nil {
		log.Fatal(err)
	}
	app := &application{
		config:        cfg,
		store:         store,
		authenticator: jwtAuthenticator,
		mailer:        mailer,
	}

	mux := app.mount()
	if err := app.run(mux); err != nil {
		log.Fatal(err)
	}
}
