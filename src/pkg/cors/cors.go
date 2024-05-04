package cors

import (
	"net/http"
	"strings"

	"questspace/pkg/transport"
)

type cors struct {
	allowAllOrigins  bool
	allowOrigins     []string
	normalHeaders    http.Header
	preflightHeaders http.Header
	wildcardOrigins  [][]string
}

var (
	DefaultSchemas = []string{
		"http://",
		"https://",
	}
)

func newCors(config *Config) *cors {
	if err := config.Validate(); err != nil {
		panic(err.Error())
	}

	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			config.AllowAllOrigins = true
		}
	}

	return &cors{
		allowAllOrigins:  config.AllowAllOrigins,
		allowOrigins:     normalize(config.AllowOrigins),
		normalHeaders:    generateNormalHeaders(config),
		preflightHeaders: generatePreflightHeaders(config),
		wildcardOrigins:  config.parseWildcardRules(),
	}
}

func (c *cors) middleware() transport.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if len(origin) == 0 {
				// request is not a CORS request
				next.ServeHTTP(w, r)
				return
			}
			host := r.Host

			if origin == "http://"+host || origin == "https://"+host {
				// request is not a CORS request but have origin header.
				// for example, use fetch api
				next.ServeHTTP(w, r)
				return
			}

			if !c.validateOrigin(origin) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			if !c.allowAllOrigins {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			if r.Method == http.MethodOptions {
				c.handlePreflight(w)
				w.WriteHeader(http.StatusNoContent)
				return
			}

			c.handleNormal(w)
			next.ServeHTTP(w, r)
		})
	}
}

func (c *cors) validateWildcardOrigin(origin string) bool {
	for _, w := range c.wildcardOrigins {
		if w[0] == "*" && strings.HasSuffix(origin, w[1]) {
			return true
		}
		if w[1] == "*" && strings.HasPrefix(origin, w[0]) {
			return true
		}
		if strings.HasPrefix(origin, w[0]) && strings.HasSuffix(origin, w[1]) {
			return true
		}
	}

	return false
}

func (c *cors) validateOrigin(origin string) bool {
	if c.allowAllOrigins {
		return true
	}
	for _, value := range c.allowOrigins {
		if value == origin {
			return true
		}
	}
	if len(c.wildcardOrigins) > 0 && c.validateWildcardOrigin(origin) {
		return true
	}
	return false
}

func (c *cors) handlePreflight(w http.ResponseWriter) {
	header := w.Header()
	for key, value := range c.preflightHeaders {
		header[key] = value
	}
}

func (c *cors) handleNormal(w http.ResponseWriter) {
	header := w.Header()
	for key, value := range c.normalHeaders {
		header[key] = value
	}
}
