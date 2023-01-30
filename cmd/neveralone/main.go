package main

import (
	"context"
	"fmt"
	"github.com/julez-dev/neveralone/internal/cli"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	app := cli.New(
		os.Stderr,
		os.Stdin,
		os.Args,
		Version,
		Commit,
		Date,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := app.Run(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
