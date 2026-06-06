package main

import (
	"log/slog"
	"os"

	"github.com/adam-alberty/dnsaur/internal/cli"
)

func main() {
	if err := cli.Run(); err != nil {
		slog.Error("command failed", "err", err)
		os.Exit(1)
	} else {
		slog.Info("exitting")
		os.Exit(0)

	}
}
