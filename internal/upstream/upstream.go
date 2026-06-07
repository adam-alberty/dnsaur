package upstream

import (
	"errors"
	"net/url"
	"sync/atomic"
	"time"
)

type Upstream struct {
	addresses []url.URL
	nextIndex atomic.Uint32
	timeoutMs time.Duration
}

func New(urls []url.URL, timeoutMs time.Duration) (*Upstream, error) {
	if len(urls) == 0 {
		return nil, errors.New("there needs to be at least 1 upstream server")
	}

	upstream := &Upstream{
		addresses: urls,
		timeoutMs: timeoutMs,
	}

	return upstream, nil
}

func (u *Upstream) Next() *url.URL {
	idx := u.nextIndex.Add(1)
	return &u.addresses[idx%uint32(len(u.addresses))]
}

func (u *Upstream) GetTimeout() time.Duration {
	return u.timeoutMs
}
