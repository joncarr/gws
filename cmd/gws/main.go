package main

import (
	"context"
	"os"

	"github.com/joncarr/gws/pkg/gws"
)

func main() {
	code, err := gws.Execute(context.Background(), os.Args[1:], gws.Options{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
	if err != nil {
		os.Exit(code)
	}
	os.Exit(code)
}
