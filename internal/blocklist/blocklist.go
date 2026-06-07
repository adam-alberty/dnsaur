package blocklist

import (
	"bufio"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"

	"codeberg.org/miekg/dns/dnsutil"
)

// Holds blocked domains.
type BlockList struct {
	enabled bool
	domains atomic.Pointer[map[string]struct{}]
}

// Loads blocklists.
func Load(enabled bool, lists []string) (*BlockList, error) {
	if !enabled {
		slog.Warn("domain blocking is disabled")
	}

	blocklist := &BlockList{
		enabled: enabled,
	}
	domains := make(map[string]struct{})
	blocklist.domains.Store(&domains)

	for _, source := range lists {
		count, err := blocklist.Add(source)
		if err != nil {
			return nil, err
		}
		slog.Info("loaded blocklist", "source", source, slog.Int("domain_count", count))
	}

	return blocklist, nil
}

func (bl *BlockList) IsEnabled() bool {
	return bl.enabled
}

// Checks if domain is in the blocklist.
func (bl *BlockList) Contains(domain string) bool {
	domain = strings.ToLower(dnsutil.Fqdn(domain))
	dPtr := bl.domains.Load()
	if dPtr == nil {
		return false
	}
	_, ok := (*dPtr)[domain]
	return ok
}

// Expands blocklist.
func (bl *BlockList) Add(source string) (int, error) {
	dPtr := bl.domains.Load()
	newDomains := make(map[string]struct{})
	if dPtr != nil {
		for k, v := range *dPtr {
			newDomains[k] = v
		}
	}

	var scanner *bufio.Scanner
	if isURL(source) {
		resp, err := http.Get(source)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		scanner = bufio.NewScanner(resp.Body)
	} else {
		file, err := os.Open(source)
		if err != nil {
			return 0, err
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)

	}

	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)

		switch len(fields) {
		case 0:
			continue
		case 1:
			line = fields[0]
		default:
			line = fields[len(fields)-1]
		}

		line = strings.ToLower(dnsutil.Fqdn(line))

		newDomains[line] = struct{}{}
		count += 1
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	bl.domains.Store(&newDomains)

	return count, nil
}

// Checks if blocklist source is URL.
func isURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	return u.Scheme != "" && u.Host != ""
}
