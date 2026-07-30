package gateway

import (
	"hash/fnv"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type backend struct {
	url    *url.URL
	active atomic.Int64
	proxy  *httputil.ReverseProxy
}

type balancer struct {
	backends []*backend
	strategy string
	rr       atomic.Uint64
	mu       sync.Mutex
}

func newBalancer(raw []string, strategy string) (*balancer, error) {
	result := &balancer{strategy: strategy}
	for _, value := range raw {
		u, err := url.Parse(value)
		if err != nil {
			return nil, err
		}
		proxy := httputil.NewSingleHostReverseProxy(u)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = u.Host
		}
		proxy.ErrorHandler = func(writer http.ResponseWriter, _ *http.Request, err error) {
			log.Printf("proxy_error backend=%s error=%v", u, err)
			http.Error(writer, "upstream unavailable", http.StatusBadGateway)
		}
		result.backends = append(result.backends, &backend{url: u, proxy: proxy})
	}
	return result, nil
}

func (b *balancer) pick(key string) *backend {
	if len(b.backends) == 1 {
		return b.backends[0]
	}
	switch b.strategy {
	case "least_connections":
		best := b.backends[0]
		for _, candidate := range b.backends[1:] {
			if candidate.active.Load() < best.active.Load() {
				best = candidate
			}
		}
		return best
	case "consistent_hash":
		h := fnv.New32a()
		_, _ = h.Write([]byte(key))
		return b.backends[uint64(h.Sum32())%uint64(len(b.backends))]
	default:
		return b.backends[b.rr.Add(1)%uint64(len(b.backends))]
	}
}
