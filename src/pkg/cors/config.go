package cors

import (
	"strings"

	"golang.org/x/xerrors"
)

type Config struct {
	AllowAllOrigins bool     `yaml:"allow-all-origins"`
	AllowOrigins    []string `yaml:"allow-origins"`
	AllowHeaders    []string `yaml:"allow-headers"`
	AllowMethods    []string `yaml:"allow-methods"`
}

func (c *Config) AddAllowMethods(methods ...string) {
	c.AllowMethods = append(c.AllowMethods, methods...)
}

func (c *Config) AddAllowHeaders(headers ...string) {
	c.AllowHeaders = append(c.AllowHeaders, headers...)
}

func (c *Config) getAllowedSchemas() []string {
	return DefaultSchemas
}

func (c *Config) validateAllowedSchemas(origin string) bool {
	allowedSchemas := c.getAllowedSchemas()
	for _, schema := range allowedSchemas {
		if strings.HasPrefix(origin, schema) {
			return true
		}
	}
	return false
}

func (c *Config) Validate() error {
	if c.AllowAllOrigins && len(c.AllowOrigins) > 0 {
		return xerrors.New("conflict settings: all origins enabled. AllowOrigins is not needed")
	}
	if !c.AllowAllOrigins && len(c.AllowOrigins) == 0 {
		return xerrors.New("conflict settings: all origins disabled")
	}
	for _, origin := range c.AllowOrigins {
		if !strings.Contains(origin, "*") && !c.validateAllowedSchemas(origin) {
			return xerrors.New("bad origin: origins must contain '*' or include " + strings.Join(c.getAllowedSchemas(), ","))
		}
	}
	return nil
}

func (c *Config) parseWildcardRules() [][]string {
	var wRules [][]string //nolint:prealloc

	for _, o := range c.AllowOrigins {
		if !strings.Contains(o, "*") {
			continue
		}

		if c := strings.Count(o, "*"); c > 1 {
			panic("only one * is allowed")
		}

		i := strings.Index(o, "*")
		if i == 0 {
			wRules = append(wRules, []string{"*", o[1:]})
			continue
		}
		if i == (len(o) - 1) {
			wRules = append(wRules, []string{o[:i], "*"})
			continue
		}

		wRules = append(wRules, []string{o[:i], o[i+1:]})
	}

	return wRules
}
