package httpapi

import (
	"net/http"
	"time"

	"timesync/backend/internal/mailer"
	"timesync/backend/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

const (
	accessTTL        = 30 * time.Minute
	refreshTTL       = 30 * 24 * time.Hour
	codeTTL          = 10 * time.Minute
	refreshGrace     = 30 * time.Second
	requestCodeLimit = 3
	verifyCodeLimit  = 5
)

type API struct {
	store      *store.Store
	mailer     mailer.Mailer
	clock      func() time.Time
	emailLimit *attemptTracker
	failLimit  *attemptTracker
}

func New(store *store.Store, mailer mailer.Mailer) *API {
	return &API{
		store:      store,
		mailer:     mailer,
		clock:      time.Now,
		emailLimit: newAttemptTracker(),
		failLimit:  newAttemptTracker(),
	}
}

func (a *API) Handler() http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(15 * time.Second))

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	router.Route("/auth", func(r chi.Router) {
		r.With(httprate.Limit(10, time.Hour, httprate.WithKeyFuncs(httprate.KeyByIP))).
			Post("/request-code", a.handleRequestCode)

		r.With(httprate.Limit(20, time.Hour, httprate.WithKeyFuncs(httprate.KeyByIP))).
			Post("/verify-code", a.handleVerifyCode)

		r.With(httprate.Limit(10, time.Minute, httprate.WithKeyFuncs(keyByDeviceID))).
			Post("/refresh", a.handleRefresh)

		r.Post("/logout", a.handleLogout)
	})

	return router
}
