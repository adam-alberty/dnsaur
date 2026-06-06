package resolver

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/adam-alberty/dnsaur/internal/blocklist"
	"github.com/adam-alberty/dnsaur/internal/cache"
	"github.com/adam-alberty/dnsaur/internal/config"
)

type Resolver struct {
	// config
	cfg *config.Config
	// cache
	cache *cache.Cache
	// upstream
	nextUpstreamIdx atomic.Uint64
	// blocklist
	bl *blocklist.BlockList
}

func New(cfg *config.Config) (*Resolver, error) {
	if !cfg.Blocking.Enabled {
		slog.Warn("domain blocking is disabled")
	}

	// Set up blocklist
	bl := blocklist.New()
	for _, source := range cfg.Blocking.Lists {
		count, err := bl.Add(source)
		if err != nil {
			return nil, err
		}
		slog.Info("loaded blocklist", "source", source, slog.Int("domain_count", count))
	}

	resolver := Resolver{
		cfg:   cfg,
		bl:    bl,
		cache: cache.New(),
	}
	return &resolver, nil
}

func (r *Resolver) nextUpstream() config.Upstream {
	upstreams := r.cfg.Upstream
	idx := r.nextUpstreamIdx.Add(1)
	return upstreams[idx%uint64(len(upstreams))]
}

func (resolver *Resolver) HandleDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) {
	m := r.Copy()

	// query should have exactly 1 question
	if len(m.Question) != 1 {
		slog.Debug("DNS request should have exactly 1 question", slog.Int("question", len(m.Question)))
		dnsutil.SetReply(m, r)
		m.Answer = nil
		m.Rcode = dns.RcodeFormatError
		m.Pack()
		io.Copy(w, m)

		return
	}

	key := cache.Key(m.Question[0])

	// look up cache
	if cached, ok := resolver.cache.Get(key); ok {
		slog.Debug("cache hit", "domain", m.Question[0].Header().Name)
		cached.ID = m.ID
		cached.Pack()
		io.Copy(w, cached)

		return
	}

	// check blocklist
	if resolver.cfg.Blocking.Enabled {
		for _, q := range m.Question {
			// check if blocked
			if resolver.bl.Contains(q.Header().Name) {
				slog.Debug("domain blocked", "domain", q.Header().Name)

				dnsutil.SetReply(m, r)
				m.Answer = nil
				m.Rcode = dns.RcodeNameError
				m.Authoritative = true
				m.Pack()

				io.Copy(w, m)

				return
			}
		}
	}

	// get reply from upstream
	// TODO reuse clients between requests
	client := dns.NewClient()

	upstream := resolver.nextUpstream()

	queryCtx, cancel := context.WithTimeout(ctx, time.Duration(upstream.TimeoutMs)*time.Millisecond)
	defer cancel()

	upstreamResponse, duration, err := client.Exchange(queryCtx, m, upstream.Address.URL.Scheme, upstream.Address.URL.Host)
	if err != nil {
		slog.Error("invalid upstream reply", "err", err)
		dnsutil.SetReply(m, r)
		m.Answer = nil
		m.Rcode = dns.RcodeServerFailure
		io.Copy(w, m)

		return
	}
	slog.Debug("fetching reply", slog.String("domain", m.Question[0].Header().Name), slog.String("upstream_url", upstream.Address.URL.String()), slog.Duration("duration", duration))

	// cache the response
	slog.Debug("caching upstream response", "domain", m.Question[0].Header().Name)
	resolver.cache.Set(
		key,
		upstreamResponse,
		time.Duration(minTTL(upstreamResponse))*time.Second,
	)

	io.Copy(w, upstreamResponse)
}

func minTTL(msg *dns.Msg) uint32 {
	var min uint32 = 0

	check := func(rrs []dns.RR) {
		for _, rr := range rrs {
			if rr == nil {
				continue
			}

			ttl := rr.Header().TTL

			if min == 0 || ttl < min {
				min = ttl
			}
		}
	}

	check(msg.Answer)
	check(msg.Ns)
	check(msg.Extra)

	return min
}
