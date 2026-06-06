package cli

import (
	"context"
	"log/slog"

	"github.com/adam-alberty/dnsaur/internal/config"
	"github.com/adam-alberty/dnsaur/internal/logging"
	"github.com/adam-alberty/dnsaur/internal/server"
	"github.com/urfave/cli/v3"
)

// Runs a server
func start(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")

	// parse config
	cfg, err := config.ParseConfig(configPath)
	if err != nil {
		return err
	}

	// set up logger
	logging.SetupLogger(cfg.Logging)

	slog.Info("loaded config", "config_path", configPath)

	server, err := server.NewServer(cfg)
	if err != nil {
		return err
	}

	return server.Run(ctx)
}
