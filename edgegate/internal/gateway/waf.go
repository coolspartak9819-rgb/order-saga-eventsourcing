package gateway

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var defaultWAFPatterns = []string{
	`(?i)(union\s+select|select\s+.+\s+from|insert\s+into|drop\s+table|--\s|/\*.*\*/)`,
	`(?i)(<script\b|javascript:|onerror\s*=|onload\s*=)`,
	`(?i)(\.\./\.\./|%2e%2e%2f|\x00)`,
}

type waf struct{ patterns []*regexp.Regexp }

func newWAF(extra []string) (*waf, error) {
	result := &waf{}
	for _, expression := range append(defaultWAFPatterns, extra...) {
		compiled, err := regexp.Compile(expression)
		if err != nil {
			return nil, fmt.Errorf("compile WAF pattern: %w", err)
		}
		result.patterns = append(result.patterns, compiled)
	}
	return result, nil
}

func (w *waf) inspect(r *http.Request) (string, error) {
	parts := []string{r.URL.RequestURI()}
	if decoded, err := url.QueryUnescape(r.URL.RequestURI()); err == nil {
		parts = append(parts, decoded)
	}
	for key, values := range r.Header {
		parts = append(parts, key+":"+strings.Join(values, ","))
	}
	if r.Body != nil && r.ContentLength != 0 {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			return "", err
		}
		r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(body), r.Body))
		parts = append(parts, string(body))
	}
	request := strings.Join(parts, "\n")
	for _, pattern := range w.patterns {
		if pattern.MatchString(request) {
			return pattern.String(), nil
		}
	}
	return "", nil
}
