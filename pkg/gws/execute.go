package gws

import (
	"context"
	"io"

	"github.com/joncarr/gws/internal/app"
)

type Options struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	ConfigPath string
}

func Execute(ctx context.Context, args []string, opts Options) (int, error) {
	return app.Execute(ctx, args, app.Options{
		Stdin:      opts.Stdin,
		Stdout:     opts.Stdout,
		Stderr:     opts.Stderr,
		ConfigPath: opts.ConfigPath,
	})
}
