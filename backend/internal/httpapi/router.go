package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"timesync/backend/internal/mailer"
	"timesync/backend/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

type Settings struct {
	AccessTTL              time.Duration
	RefreshTTL             time.Duration
	CodeTTL                time.Duration
	RefreshGrace           time.Duration
	TeamSizeLimit          int
	RequestCodeEmailLimit  int
	RequestCodeEmailWindow time.Duration
	RequestCodeIPLimit     int
	RequestCodeIPWindow    time.Duration
	VerifyCodeEmailLimit   int
	VerifyCodeEmailWindow  time.Duration
	VerifyCodeLock         time.Duration
	VerifyCodeIPLimit      int
	VerifyCodeIPWindow     time.Duration
	RefreshDeviceLimit     int
	RefreshDeviceWindow    time.Duration
}

type API struct {
	store      *store.Store
	mailer     mailer.Mailer
	logger     *slog.Logger
	settings   Settings
	clock      func() time.Time
	emailLimit *attemptTracker
	failLimit  *attemptTracker
}

func New(store *store.Store, mailer mailer.Mailer, settings Settings, logger *slog.Logger) *API {
	if logger == nil {
		logger = slog.Default()
	}
	return &API{
		store:      store,
		mailer:     mailer,
		logger:     logger,
		settings:   settings,
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
		r.With(httprate.Limit(a.settings.RequestCodeIPLimit, a.settings.RequestCodeIPWindow, httprate.WithKeyFuncs(httprate.KeyByIP))).
			Post("/request-code", a.handleRequestCode)

		r.With(httprate.Limit(a.settings.VerifyCodeIPLimit, a.settings.VerifyCodeIPWindow, httprate.WithKeyFuncs(httprate.KeyByIP))).
			Post("/verify-code", a.handleVerifyCode)

		r.With(httprate.Limit(a.settings.RefreshDeviceLimit, a.settings.RefreshDeviceWindow, httprate.WithKeyFuncs(keyByDeviceID))).
			Post("/refresh", a.handleRefresh)

		r.Post("/logout", a.handleLogout)
	})

	return router
}
