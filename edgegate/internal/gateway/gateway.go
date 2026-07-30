package gateway

import (
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coolspartak9819-rgb/edgegate/internal/config"
	"github.com/redis/go-redis/v9"
)

type Gateway struct {
	config  atomic.Pointer[runtimeConfig]
	plugins *pluginRegistry
	redis   *redis.Client
}

type runtimeConfig struct {
	routes []*runtimeRoute
}

type runtimeRoute struct {
	path     string
	balancer *balancer
	handler  http.Handler
}

func New(cfg config.Config) (*Gateway, error) {
	gateway := &Gateway{
		plugins: newPluginRegistry(),
		redis:   redis.NewClient(&redis.Options{Addr: cfg.RedisAddr}),
	}
	if err := gateway.Apply(cfg); err != nil {
		_ = gateway.redis.Close()
		return nil, err
	}
	return gateway, nil
}

func (g *Gateway) Apply(cfg config.Config) error {
	if err := config.Validate(cfg); err != nil {
		return err
	}
	runtime := &runtimeConfig{}
	for _, routeCfg := range cfg.Routes {
		balancer, err := newBalancer(routeCfg.Backends, strings.ToLower(defaultString(routeCfg.Strategy, "round_robin")))
		if err != nil {
			return err
		}
		route := &runtimeRoute{path: routeCfg.Path, balancer: balancer}
		var base http.Handler = http.HandlerFunc(route.proxy)
		if routeCfg.WAF != nil && routeCfg.WAF.Enabled {
			waf, err := newWAF(routeCfg.WAF.ExtraPatterns)
			if err != nil {
				return err
			}
			base = func(next http.Handler) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					if matched, err := waf.inspect(r); err != nil {
						http.Error(w, "waf inspection failed", http.StatusBadRequest)
						return
					} else if matched != "" {
						log.Printf("waf_block path=%s rule=%q", r.URL.Path, matched)
						http.Error(w, "request blocked by WAF", http.StatusForbidden)
						return
					}
					next.ServeHTTP(w, r)
				}
			}(base)
		}
		if routeCfg.RateLimit != nil {
			limiter := newLimiter(g.redis, routeCfg.RateLimit.RequestsPerSecond, routeCfg.RateLimit.Burst)
			base = func(next http.Handler) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					allowed, err := limiter.allow(r.Context(), clientKey(r))
					if err != nil {
						http.Error(w, "rate limiter unavailable", http.StatusServiceUnavailable)
						return
					}
					if !allowed {
						http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
						return
					}
					next.ServeHTTP(w, r)
				}
			}(base)
		}
		chained, err := g.plugins.chain(routeCfg.Plugins, base)
		if err != nil {
			return err
		}
		route.handler = chained
		runtime.routes = append(runtime.routes, route)
	}
	sort.SliceStable(runtime.routes, func(i, j int) bool {
		return len(runtime.routes[i].path) > len(runtime.routes[j].path)
	})
	g.config.Store(runtime)
	return nil
}

func (g *Gateway) Close() error { return g.redis.Close() }

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	runtime := g.config.Load()
	for _, route := range runtime.routes {
		if strings.HasPrefix(r.URL.Path, route.path) {
			route.handler.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func (r *runtimeRoute) proxy(w http.ResponseWriter, request *http.Request) {
	backend := r.balancer.pick(clientKey(request))
	backend.active.Add(1)
	defer backend.active.Add(-1)
	backend.proxy.ServeHTTP(w, request)
}

func (g *Gateway) Reload(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	return g.Apply(cfg)
}

func WatchConfig(path string, gateway *Gateway, interval time.Duration, stop <-chan struct{}) {
	var modified time.Time
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			info, err := os.Stat(path)
			if err != nil || !info.ModTime().After(modified) {
				continue
			}
			if err := gateway.Reload(path); err != nil {
				log.Printf("config reload failed: %v", err)
				continue
			}
			modified = info.ModTime()
			log.Printf("config reloaded: %s", path)
		}
	}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
