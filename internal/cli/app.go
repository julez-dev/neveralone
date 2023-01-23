package cli

import (
	"context"
	"fmt"
	"github.com/julez-dev/neveralone/internal/handler"
	"github.com/julez-dev/neveralone/internal/rest"
	"github.com/julez-dev/neveralone/internal/store"
	"github.com/rs/zerolog"
	"github.com/urfave/cli/v3"
	"io"
	"net/mail"
	"runtime"
	"time"
)

type App struct {
	w    io.Writer
	r    io.Reader
	args []string

	version string
	commit  string
	date    string
}

func New(w io.Writer, r io.Reader, args []string, version string, commit string, date string) *App {
	return &App{
		w:       w,
		r:       r,
		args:    args,
		version: version,
		commit:  commit,
		date:    date,
	}
}

func (a *App) Run(ctx context.Context) error {
	humanOutput := zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.Out = a.w
			w.TimeFormat = time.RFC3339
		},
	)

	logger := zerolog.New(humanOutput).With().
		Timestamp().
		Str("version", a.version).
		Str("commit", a.commit).
		Logger()

	_ = logger

	app := &cli.App{
		Name:   "neveralone",
		Usage:  "Watch videos with friends",
		Writer: a.w,
		Reader: a.r,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "addr",
				Usage:   "HTTP Server Address",
				Value:   ":8080",
				EnvVars: []string{"HTTP_ADDR"},
			},
		},
		Authors: []any{&mail.Address{Name: "julez-dev", Address: "julez-dev@pm.me"}},
		Commands: []*cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Print the version",
				Action: func(c *cli.Context) error {
					res := fmt.Sprintf("neveralone version %s\n"+
						"commit: %s\n"+
						"built at: %s\n"+
						"goos: %s\n"+
						"goarch: %s\n"+
						"go version: %s\n",
						a.version, a.commit, a.date, runtime.GOOS, runtime.GOARCH, runtime.Version(),
					)

					if _, err := io.WriteString(a.w, res); err != nil {
						return err
					}

					return nil
				},
			},
		},
		Action: func(c *cli.Context) error {
			userStore := store.NewUser()
			sessionStore := store.NewSession()

			tmplHandler, err := handler.NewTemplate(sessionStore)

			if err != nil {
				return err
			}

			api := rest.New(
				&rest.Config{HostAndPort: c.String("addr")},
				logger,
				tmplHandler,
				userStore,
				sessionStore,
				tmplHandler,
			)

			return api.Launch(c.Context)
		},
	}

	return app.RunContext(ctx, a.args)
}
