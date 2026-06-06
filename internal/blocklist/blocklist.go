package blocklist

import (
	"bufio"
	"net/http"
	"net/url"
	"os"
	"strings"

	"codeberg.org/miekg/dns/dnsutil"
)

// Holds blocked domains.
type BlockList struct {
	domains map[string]struct{}
}

func New() *BlockList {
	return &BlockList{
		domains: make(map[string]struct{}),
	}
}

// Checks if domain is in the blocklist.
func (bl *BlockList) Contains(domain string) bool {
	domain = strings.ToLower(dnsutil.Fqdn(domain))
	_, ok := bl.domains[domain]
	return ok
}

// Expands blocklist.
func (bl *BlockList) Add(source string) (int, error) {
	newDomains := bl.domains
	if newDomains == nil {
		newDomains = make(map[string]struct{})
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

	bl.domains = newDomains

	return count, scanner.Err()
}

// Checks if blocklist source is URL.
func isURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}

	return u.Scheme != "" && u.Host != ""
}
