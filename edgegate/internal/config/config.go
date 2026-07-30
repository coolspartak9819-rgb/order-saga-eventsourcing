package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	Listen    string  `json:"listen"`
	RedisAddr string  `json:"redis_addr"`
	Routes    []Route `json:"routes"`
}

type Route struct {
	Path      string           `json:"path"`
	Backends  []string         `json:"backends"`
	Strategy  string           `json:"strategy"`
	Plugins   []string         `json:"plugins"`
	RateLimit *RateLimitConfig `json:"rate_limit,omitempty"`
	WAF       *WAFConfig       `json:"waf,omitempty"`
}

type RateLimitConfig struct {
	RequestsPerSecond float64 `json:"requests_per_second"`
	Burst             int     `json:"burst"`
}

type WAFConfig struct {
	Enabled       bool     `json:"enabled"`
	ExtraPatterns []string `json:"extra_patterns,omitempty"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if cfg.Listen == "" {
		cfg.Listen = ":8080"
	}
	if cfg.RedisAddr == "" {
		cfg.RedisAddr = "localhost:6379"
	}
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Validate(cfg Config) error {
	if len(cfg.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
	}
	seen := make(map[string]struct{}, len(cfg.Routes))
	for i, route := range cfg.Routes {
		if route.Path == "" || !strings.HasPrefix(route.Path, "/") {
			return fmt.Errorf("routes[%d].path must start with /", i)
		}
		if _, ok := seen[route.Path]; ok {
			return fmt.Errorf("duplicate route path %q", route.Path)
		}
		seen[route.Path] = struct{}{}
		if len(route.Backends) == 0 {
			return fmt.Errorf("route %q has no backends", route.Path)
		}
		for _, backend := range route.Backends {
			u, err := url.Parse(backend)
			if err != nil || u.Scheme == "" || u.Host == "" {
				return fmt.Errorf("route %q has invalid backend %q", route.Path, backend)
			}
		}
		strategy := strings.ToLower(route.Strategy)
		if strategy == "" {
			strategy = "round_robin"
		}
		if strategy != "round_robin" && strategy != "least_connections" && strategy != "consistent_hash" {
			return fmt.Errorf("route %q has unsupported strategy %q", route.Path, route.Strategy)
		}
		if route.RateLimit != nil && (route.RateLimit.RequestsPerSecond <= 0 || route.RateLimit.Burst < 1) {
			return fmt.Errorf("route %q has invalid rate limit", route.Path)
		}
	}
	return nil
}
