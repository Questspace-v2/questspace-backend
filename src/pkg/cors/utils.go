package cors

import (
	"net/http"
	"strings"
)

type converter func(string) string

func generateNormalHeaders(c *Config) http.Header {
	headers := make(http.Header)
	headers.Set("Access-Control-Allow-Credentials", "true")
	if c.AllowAllOrigins {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Vary", "Origin")
	}
	return headers
}

func generatePreflightHeaders(c *Config) http.Header {
	headers := make(http.Header)

	headers.Set("Access-Control-Allow-Credentials", "true")
	if len(c.AllowMethods) > 0 {
		allowMethods := convert(normalize(c.AllowMethods), strings.ToUpper)
		value := strings.Join(allowMethods, ",")
		headers.Set("Access-Control-Allow-Methods", value)
	}
	if len(c.AllowHeaders) > 0 {
		allowHeaders := convert(normalize(c.AllowHeaders), http.CanonicalHeaderKey)
		value := strings.Join(allowHeaders, ",")
		headers.Set("Access-Control-Allow-Headers", value)
	}

	if c.AllowAllOrigins {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Add("Vary", "Origin")
		headers.Add("Vary", "Access-Control-Request-Method")
		headers.Add("Vary", "Access-Control-Request-Headers")
	}
	return headers
}

func normalize(values []string) []string {
	if values == nil {
		return nil
	}
	distinctMap := make(map[string]bool, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		value = strings.ToLower(value)
		if _, seen := distinctMap[value]; !seen {
			normalized = append(normalized, value)
			distinctMap[value] = true
		}
	}
	return normalized
}

func convert(s []string, c converter) []string {
	out := make([]string, 0, len(s))
	for _, i := range s {
		out = append(out, c(i))
	}
	return out
}
