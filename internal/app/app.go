package app

import (
	"context"
	"io"
	"os"

	"github.com/joncarr/gws/internal/cli"
	"github.com/joncarr/gws/internal/commands"
	"github.com/joncarr/gws/internal/google"
	"github.com/joncarr/gws/internal/logging"
)

type Options struct {
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	ConfigPath string
}

func Execute(ctx context.Context, args []string, opts Options) (int, error) {
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	parsed, err := cli.Parse(args)
	if err != nil {
		logging.Error(opts.Stderr, err)
		return 2, err
	}
	configPath := opts.ConfigPath
	if parsed.Flags["config"] != "" {
		configPath = parsed.Flags["config"]
	} else if configPath == "" {
		configPath = os.Getenv("GWS_CONFIG")
	}
	runner := commands.Runner{
		Stdin:     opts.Stdin,
		Stdout:    opts.Stdout,
		Stderr:    opts.Stderr,
		Config:    configPath,
		Directory: google.AdminDirectoryClient{},
		Gmail:     google.AdminGmailClient{},
		Sheets:    google.AdminSheetsExporter{},
	}
	if err := runner.Run(ctx, parsed); err != nil {
		logging.Error(opts.Stderr, err)
		return 1, err
	}
	return 0, nil
}
