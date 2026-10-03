package ratelimit

import (
	"crypto/subtle"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ProxySecretHeader carries the secret the Next.js proxy shares with the API.
const ProxySecretHeader = "X-Proxy-Secret"

// ClientIPResolver finds the IP of the client behind a request. The API is
// called through the Next.js proxy, so the connection comes from the proxy
// and the client IP travels in the X-Real-IP header. Anyone can send that
// header, so it is only trusted when the request proves it comes from the
// proxy: either the connection comes from one of the trusted proxy networks
// or the request carries the shared proxy secret.
type ClientIPResolver struct {
	trustedProxies []netip.Prefix
	proxySecret    []byte
}

// NewClientIPResolver creates a resolver. An empty proxySecret disables the
// secret check, leaving only trustedProxies.
func NewClientIPResolver(
	trustedProxies []netip.Prefix,
	proxySecret string,
) *ClientIPResolver {
	return &ClientIPResolver{
		trustedProxies: trustedProxies,
		proxySecret:    []byte(proxySecret),
	}
}

// ClientIP returns the client IP of r, or false when the remote address
// cannot be parsed.
func (c *ClientIPResolver) ClientIP(r *http.Request) (netip.Addr, bool) {
	remote, ok := remoteAddr(r)
	if !ok {
		return netip.Addr{}, false
	}

	if !c.isTrustedProxy(remote) && !c.hasProxySecret(r) {
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

// hasProxySecret compares in constant time, so the response time does not
// reveal how much of a guessed secret is right.
func (c *ClientIPResolver) hasProxySecret(r *http.Request) bool {
	if len(c.proxySecret) == 0 {
		return false
	}

	return subtle.ConstantTimeCompare(
		[]byte(r.Header.Get(ProxySecretHeader)),
		c.proxySecret,
	) == 1
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
