package rest

import (
	"context"
	"errors"
	"github.com/julez-dev/neveralone/internal/static"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/sync/errgroup"
	"io/fs"
	"net/http"
	"time"
)

type Config struct {
	HostAndPort string
}

type Server struct {
	config *Config
	logger zerolog.Logger

	jwtGenerator jwtGenerator

	// handler
	homeHandler  homeHandler
	partyHandler partyTemplateHandler

	// store
	sessionStore sessionStore

	closeCTX context.Context
}

func New(
	config *Config,
	logger zerolog.Logger,
	homeHandler homeHandler,
	sessionStore sessionStore,
	partyHandler partyTemplateHandler,
	jwtGenerator jwtGenerator,
) *Server {
	return &Server{
		config:       config,
		logger:       logger,
		homeHandler:  homeHandler,
		sessionStore: sessionStore,
		partyHandler: partyHandler,
		jwtGenerator: jwtGenerator,
	}
}

func (s *Server) Launch(ctx context.Context) error {
	s.closeCTX = ctx

	// setup echo
	e := echo.New()
	e.Server.WriteTimeout = time.Second * 15
	e.Server.ReadTimeout = time.Second * 15
	e.Server.IdleTimeout = time.Second * 60
	e.Server.MaxHeaderBytes = 2 * 1024
	e.Logger = lecho.From(s.logger)
	e.HideBanner = true

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		data := &struct {
			Message string `json:"message"`
		}{
			Message: err.Error(),
		}

		c.JSON(http.StatusInternalServerError, data)
	}

	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return c.Request().URL.Path == "/favicon.ico"
		},
	}))
	e.Use(middleware.Recover())
	e.Use(s.getUserMiddleware)

	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "image/x-icon", static.IconFile)
	})
	e.GET("/", s.GetHome)
	e.POST("/party", s.CreateParty)
	e.GET("/party", s.CreatePartyCustom)
	e.Match([]string{http.MethodGet, http.MethodPost}, "/party/:id", s.GetParty)
	e.GET("/party/:id/ws", s.JoinWS)

	sub, err := fs.Sub(static.StaticFiles, "static")

	if err != nil {
		return err
	}

	e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", http.FileServer(http.FS(sub)))))

	wg, ctx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		<-ctx.Done()

		s.logger.Info().Msg("shutting down server gracefully")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		return e.Shutdown(ctx)
	})

	err = e.Start(s.config.HostAndPort)
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}

	if err := wg.Wait(); err != nil {
		return err
	}

	return nil
}
