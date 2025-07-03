package dynforward

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() {
	plugin.Register("dynforward", setup)
}

func setup(c *caddy.Controller) error {
	var internalDNS, externalDNS string
	for c.Next() {
		args := c.RemainingArgs()
		for i := 0; i < len(args); i = i + 2 {
			if args[i] == "internal" && i+1 < len(args) {
				internalDNS = args[i+1]
			}
			if args[i] == "external" && i+1 < len(args) {
				externalDNS = args[i+1]
			}
		}
	}
	if internalDNS == "" && externalDNS == "" {
		return plugin.Error("dynforward", c.ArgErr())
	}

	df := DynForward{
		InternalDNS: internalDNS,
		ExternalDNS: externalDNS,
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		df.Next = next
		return df
	})
	return nil
}
