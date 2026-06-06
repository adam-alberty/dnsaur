package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"
)

func Run() error {
	defaultConfigPath, err := getDefaultConfigPath()
	if err != nil {
		return err
	}

	cmd := &cli.Command{
		Name:   "dnsaur",
		Usage:  "Forwarding dns server with ad-blocking.",
		Action: start,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "config",
				Value: defaultConfigPath,
				Usage: "Path to configuration file",
			},
		},
	}

	return cmd.Run(context.Background(), os.Args)
}

func getDefaultConfigPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	defaultConfig := filepath.Join(cwd, "config.toml")

	return defaultConfig, nil
}
