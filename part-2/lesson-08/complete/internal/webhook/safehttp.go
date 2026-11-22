package webhook

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"syscall"
	"time"
)

// ErrBlocked means the URL leads to an address inside our network.
var ErrBlocked = errors.New("not a public address")

// cgnat is 100.64.0.0/10, the carrier-grade NAT range: private in practice.
var cgnat = netip.MustParsePrefix("100.64.0.0/10")

// isPublic reports whether ip is an address on the public internet.
// Unmap first: ::ffff:127.0.0.1 is 127.0.0.1 in disguise.
func isPublic(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsValid() &&
		!ip.IsLoopback() && // 127.0.0.0/8, ::1
		!ip.IsPrivate() && // 10/8, 172.16/12, 192.168/16, fc00::/7
		!ip.IsLinkLocalUnicast() && // 169.254/16 (cloud metadata!), fe80::/10
		!ip.IsUnspecified() && // 0.0.0.0, ::
		!ip.IsMulticast() && !ip.IsLinkLocalMulticast() && !ip.IsInterfaceLocalMulticast() &&
		!cgnat.Contains(ip)
}

// publicOnly runs inside the dialer after DNS and right before connect, with
// the real IP. Every spelling (0.0.0.0, [::], decimal), every DNS name, every
// redirect hop and every DNS rebinding trick ends up here as a plain IP.
func publicOnly(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	if !isPublic(ip) {
		return fmt.Errorf("%w: %s", ErrBlocked, ip.Unmap())
	}
	return nil
}

// SafeClient is an http.Client for URLs that users give us: it can only
// connect to public addresses, follows at most 3 redirects (each one checked
// again by the dialer), and gives up after 10 seconds.
func SafeClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, Control: publicOnly}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy:                 nil, // never through a proxy: the dialer must see the real target
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			MaxIdleConns:          10,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("stopped after 3 redirects")
			}
			return nil
		},
	}
}
