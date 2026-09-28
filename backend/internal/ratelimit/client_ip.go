package ratelimit

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIPResolver finds the IP of the client behind a request. The API is
// called through the Next.js proxy, so the connection comes from the proxy
// and the client IP travels in the X-Real-IP header. Anyone can send that
// header, so it is only trusted when the connection comes from one of the
// trusted proxy networks.
type ClientIPResolver struct {
	trustedProxies []netip.Prefix
}

func NewClientIPResolver(trustedProxies []netip.Prefix) *ClientIPResolver {
	return &ClientIPResolver{
		trustedProxies: trustedProxies,
	}
}

// ClientIP returns the client IP of r, or false when the remote address
// cannot be parsed.
func (c *ClientIPResolver) ClientIP(r *http.Request) (netip.Addr, bool) {
	remote, ok := remoteAddr(r)
	if !ok {
		return netip.Addr{}, false
	}

	if !c.isTrustedProxy(remote) {
		return remote, true
	}

	realIP, err := netip.ParseAddr(
		strings.TrimSpace(r.Header.Get("X-Real-IP")),
	)
	if err != nil {
		// A trusted proxy that sends no valid client IP is treated as the
		// client, so its requests share one limit instead of bypassing it.
		return remote, true
	}

	return realIP.Unmap(), true
}

func (c *ClientIPResolver) isTrustedProxy(addr netip.Addr) bool {
	for _, prefix := range c.trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// remoteAddr parses the IP of the connection, dropping the port and turning
// IPv4-mapped IPv6 addresses (::ffff:1.2.3.4) into plain IPv4. The zone is
// dropped because zoned addresses never match a prefix.
func remoteAddr(r *http.Request) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	addr, err := netip.ParseAddr(host)
	if err != nil {
		return netip.Addr{}, false
	}

	return addr.Unmap().WithZone(""), true
}
