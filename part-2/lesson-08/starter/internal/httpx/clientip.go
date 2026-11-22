package httpx

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ParseTrusted reads a comma-separated list of proxy addresses or CIDRs,
// like "10.0.0.5, 10.0.1.0/24". An empty string trusts nobody.
func ParseTrusted(list string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, s := range strings.Split(list, ",") {
		if s = strings.TrimSpace(s); s == "" {
			continue
		}
		if !strings.Contains(s, "/") {
			ip, err := netip.ParseAddr(s)
			if err != nil {
				return nil, err
			}
			out = append(out, netip.PrefixFrom(ip, ip.BitLen()))
			continue
		}
		p, err := netip.ParsePrefix(s)
		if err != nil {
			return nil, err
		}
		out = append(out, p.Masked())
	}
	return out, nil
}

// ClientIP is the address of the real client.
//
// If the connection comes from one of our trusted proxies, the client is the
// LAST entry of X-Forwarded-For: the one our proxy added. Everything to its
// left was sent by the client and proves nothing. From anyone else, the
// header is ignored and the connection's own address is the answer.
func ClientIP(r *http.Request, trusted []netip.Prefix) string {
	peer := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peer); err == nil {
		peer = host
	}
	addr, err := netip.ParseAddr(peer)
	if err != nil || !contains(trusted, addr) {
		return peer
	}
	forwarded := r.Header.Values("X-Forwarded-For")
	if len(forwarded) == 0 {
		return peer
	}
	entries := strings.Split(forwarded[len(forwarded)-1], ",")
	last := strings.TrimSpace(entries[len(entries)-1])
	if ip, err := netip.ParseAddr(last); err == nil {
		return ip.Unmap().String()
	}
	return peer
}

func contains(prefixes []netip.Prefix, ip netip.Addr) bool {
	ip = ip.Unmap()
	for _, p := range prefixes {
		if p.Contains(ip) {
			return true
		}
	}
	return false
}
