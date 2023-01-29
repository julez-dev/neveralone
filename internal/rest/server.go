package rest

import (
	"context"
	"errors"
	"github.com/julez-dev/neveralone/internal/static"
	"github.com/labstack/echo/v4"
	emiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/ziflex/lecho/v3"
	"golang.org/x/sync/errgroup"
	"io"
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

	//e.Use(middleware.OapiRequestValidator(swagger))

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		data := &struct {
			Message string `json:"message"`
		}{
			Message: err.Error(),
		}

		c.JSON(http.StatusInternalServerError, data)
	}

	e.Use(emiddleware.LoggerWithConfig(
		emiddleware.LoggerConfig{
			Skipper:          nil,
			Format:           "",
			CustomTimeFormat: "",
			CustomTagFunc:    nil,
			Output:           io.Discard,
		}))
	//e.Use(emiddleware.Recover())

	e.Use(s.getUserMiddleware)

	e.GET("/", s.GetHome)

	e.POST("/party", s.CreateParty)
	e.GET("/party/:id", s.GetParty)
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
