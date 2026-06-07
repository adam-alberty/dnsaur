package resolver

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"time"

	"codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/adam-alberty/dnsaur/internal/blocklist"
	"github.com/adam-alberty/dnsaur/internal/cache"
	"github.com/adam-alberty/dnsaur/internal/config"
	"github.com/adam-alberty/dnsaur/internal/upstream"
)

type Resolver struct {
	upstream  *upstream.Upstream
	cache     *cache.Cache
	blocklist *blocklist.BlockList
	client    *dns.Client
}

func New(cfg *config.Config) (*Resolver, error) {
	blocklist, err := blocklist.Load(cfg.Blocking.Enabled, cfg.Blocking.Lists)
	if err != nil {
		return nil, err
	}

	urls := make([]url.URL, 0, len(cfg.Upstream.Addresses))
	for _, u := range cfg.Upstream.Addresses {
		urls = append(urls, *u.URL)
	}
	upstream, err := upstream.New(urls, time.Duration(cfg.Upstream.TimeoutMs)*time.Millisecond)
	if err != nil {
		return nil, err
	}

	resolver := Resolver{
		upstream:  upstream,
		blocklist: blocklist,
		cache:     cache.New(cfg.Cache.Enabled),
		client:    dns.NewClient(),
	}
	return &resolver, nil
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

	if resolver.cache.IsEnabled() {
		key := cache.Key(m.Question[0])

		// look up cache
		if cached, ok := resolver.cache.Get(key); ok {
			slog.Debug("cache hit", "domain", m.Question[0].Header().Name)
			cached.ID = m.ID
			cached.Pack()
			io.Copy(w, cached)

			return
		}
	}

	// check blocklist
	if resolver.blocklist.IsEnabled() {
		for _, q := range m.Question {
			if resolver.blocklist.Contains(q.Header().Name) {
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

	upstream := resolver.upstream.Next()
	queryCtx, cancel := context.WithTimeout(ctx, resolver.upstream.GetTimeout())
	defer cancel()

	upstreamResponse, duration, err := resolver.client.Exchange(queryCtx, m, upstream.Scheme, upstream.Host)
	if err != nil {
		slog.Error("invalid upstream reply", "err", err)
		dnsutil.SetReply(m, r)
		m.Answer = nil
		m.Rcode = dns.RcodeServerFailure
		io.Copy(w, m)

		return
	}
	slog.Debug("fetched reply from upstream", slog.String("domain", m.Question[0].Header().Name), slog.String("upstream_url", upstream.String()), slog.Duration("duration", duration))

	if resolver.cache.IsEnabled() {
		key := cache.Key(m.Question[0])
		// cache the response
		slog.Debug("caching upstream response", "domain", m.Question[0].Header().Name)
		resolver.cache.Set(
			key,
			upstreamResponse,
			time.Duration(minTTL(upstreamResponse))*time.Second,
		)

	}

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
