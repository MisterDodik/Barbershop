package main

import (
	"log"
	"net/http"
	"time"

	"github.com/MisterDodik/Barbershop/internal/auth"
	"github.com/MisterDodik/Barbershop/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type application struct {
	config        config
	store         store.Storage
	authenticator auth.Authenticator
}
type config struct {
	addr string
	db   dbConfig
	auth authConfig
}
type authConfig struct {
	basic basicConfig
	token tokenConfig
}
type tokenConfig struct {
	secret  string
	expDate time.Duration
	iss     string
}
type basicConfig struct { //moze za neke odredjene stranice, npr admin ili tako nesto
	username string
	password string
}
type dbConfig struct {
	addr         string
	maxOpenConns int
	maxIdleConns int
	maxIdleTime  string
}

func (app *application) mount() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:3000",                     // for local dev
			"https://your-vercel-deployment.vercel.app", // for production
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", app.getHealthHandler)

		r.Route("/appointment", func(r chi.Router) {
			r.Post("/get_available_dates", app.getAvailableDates) //prilikom loadanja sajta uzeti da je selectedday = null, a to ce automatski biti danasnji dan

			//authenticated endpoints
			r.Route("/", func(r chi.Router) {
				r.Use(app.TokenAuthMiddleware)

				r.Post("/book/{slotID}", app.bookAppointment)
			})
		})

		r.Route("/authentication", func(r chi.Router) {
			r.Post("/user", app.registerUserHandler)
			r.Post("/token", app.createTokenHandler)
		})
		// r.With(paginate).Get("/", listArticles)                           // GET /articles
		// r.Get("/search", searchArticles)                                  // GET /articles/search
		// // Subrouters:
		// r.Route("/{articleID}", func(r chi.Router) {
		// 	//r.Use(ArticleCtx)
		// 	//r.Get("/", getArticle)       // GET /articles/123
		// })
	})

	return r
}

func (app *application) run(mux http.Handler) error {
	srv := &http.Server{
		Addr:         app.config.addr,
		Handler:      mux,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	log.Printf("server started at %s", app.config.addr)
	return srv.ListenAndServe()
}
