package config

import (
	"fmt"
	"net/url"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Version  int            `toml:"version"`
	Server   ServerConfig   `toml:"server"`
	Logging  LoggingConfig  `toml:"logging"`
	Cache    CacheConfig    `toml:"cache"`
	Blocking BlockingConfig `toml:"blocking"`
	Upstream Upstream       `toml:"upstream"`
}

type ServerConfig struct {
	UDPListen []string `toml:"udp_listen"`
	TCPListen []string `toml:"tcp_listen"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

type CacheConfig struct {
	Enabled    bool `toml:"enabled"`
	MaxEntries int  `toml:"max_entries"`
}

type BlockingConfig struct {
	Enabled bool     `toml:"enabled"`
	Lists   []string `toml:"lists"`
}

type Upstream struct {
	Addresses []UpstreamAddress `toml:"addresses"`
	TimeoutMs int               `toml:"timeout_ms"`
}

type UpstreamAddress struct {
	URL *url.URL
}

func (a *UpstreamAddress) UnmarshalText(text []byte) error {
	u, err := url.Parse(string(text))
	if err != nil {
		return err
	}

	switch u.Scheme {
	// TODO implement all of these
	case "udp", "tcp", "tls", "doh":
	default:
		return fmt.Errorf("invalid upstream scheme: %s", u.Scheme)
	}

	a.URL = u
	return nil
}

func ParseConfig(path string) (*Config, error) {
	config := new(Config)

	if _, err := toml.DecodeFile(path, config); err != nil {
		return nil, err
	}

	return config, nil
}
