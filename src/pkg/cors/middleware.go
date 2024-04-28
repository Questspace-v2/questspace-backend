package cors

import (
	"questspace/pkg/transport"
)

func Middleware(config *Config) transport.Middleware {
	c := newCors(config)
	return c.middleware()
}
