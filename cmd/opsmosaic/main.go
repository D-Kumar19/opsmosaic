package main

import (
	"context"
	"fmt"
	"os"

	"opsmosaic/internal/brief"
)

func main() {
	app, err := brief.NewApp()
	if err == nil {
		err = app.Run(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "OpsMosaic:", err)
		os.Exit(1)
	}
}
