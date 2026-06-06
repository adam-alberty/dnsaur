package server

import (
	"context"
	"log/slog"

	"codeberg.org/miekg/dns"
	"github.com/adam-alberty/dnsaur/internal/config"
	"github.com/adam-alberty/dnsaur/internal/resolver"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	cfg      *config.Config
	resolver *resolver.Resolver
}

func NewServer(cfg *config.Config) (*Server, error) {
	resolver, err := resolver.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Server{cfg: cfg, resolver: resolver}, nil
}

func (s *Server) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)

	// Bind to configured UDP interfaces
	for _, listenAddr := range s.cfg.Server.UDPListen {
		addr := listenAddr
		g.Go(func() error {
			return s.startUDPServer(ctx, addr)
		})
	}
	// Bind to configured TCP interfaces
	for _, listenAddr := range s.cfg.Server.TCPListen {
		addr := listenAddr
		g.Go(func() error {
			return s.startTCPServer(ctx, addr)
		})
	}

	return g.Wait()
}

func (s *Server) startUDPServer(_ context.Context, address string) error {
	mux := dns.NewServeMux()

	// resolver function
	mux.HandleFunc(".", s.resolver.HandleDNS)

	dnsServer := &dns.Server{
		Addr:    address,
		Net:     "udp",
		Handler: mux,
	}

	slog.Debug("starting UDP server", "addr", address)
	return dnsServer.ListenAndServe()
}

func (s *Server) startTCPServer(_ context.Context, address string) error {
	mux := dns.NewServeMux()

	// resolver function
	mux.HandleFunc(".", s.resolver.HandleDNS)

	dnsServer := &dns.Server{
		Addr:    address,
		Net:     "tcp",
		Handler: mux,
	}

	slog.Debug("starting TCP server", "addr", address)
	return dnsServer.ListenAndServe()
}
