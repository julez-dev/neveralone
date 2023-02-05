package main

import (
	"context"
	"github.com/julez-dev/neveralone/internal/cli"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	app := cli.New(
		os.Stdout,
		os.Stdin,
		os.Args,
		Version,
		Commit,
		Date,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		os.Exit(1)
	}
}
