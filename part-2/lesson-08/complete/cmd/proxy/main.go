// Command proxy is a small reverse proxy in front of several copies of the
// API: round-robin, X-Forwarded-For like nginx, and a cap on requests in
// flight per client IP (the edge limit lesson 1 promised).
//
//	go run ./cmd/proxy -backends http://127.0.0.1:8081,http://127.0.0.1:8082,http://127.0.0.1:8083
//
// In production you'd use nginx, Caddy, HAProxy or a cloud load balancer;
// this one is small enough to read.
package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:8000", "address to listen on")
	backends := flag.String("backends", "http://127.0.0.1:8081,http://127.0.0.1:8082,http://127.0.0.1:8083",
		"comma-separated API addresses")
	perIP := flag.Int("per-ip", 50, "most requests in flight per client IP")
	flag.Parse()

	var targets []*url.URL
	for _, b := range strings.Split(*backends, ",") {
		u, err := url.Parse(strings.TrimSpace(b))
		if err != nil {
			log.Fatal(err)
		}
		targets = append(targets, u)
	}
	var next atomic.Uint64
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(targets[next.Add(1)%uint64(len(targets))]) // round-robin
			// Like nginx's $proxy_add_x_forwarded_for: keep what the client sent,
			// then append the real address. Only that LAST entry is ours.
			pr.Out.Header["X-Forwarded-For"] = pr.In.Header["X-Forwarded-For"]
			pr.SetXForwarded()
		},
	}

	srv := &http.Server{
		Addr:              *listen,
		Handler:           capPerIP(rp, *perIP),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      40 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("proxy on http://%s -> %s", *listen, *backends)
	log.Fatal(srv.ListenAndServe())
}

// capPerIP answers 429 when one client IP already has max requests in flight.
func capPerIP(next http.Handler, max int) http.Handler {
	var mu sync.Mutex
	inFlight := map[string]int{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		mu.Lock()
		if inFlight[ip] >= max {
			mu.Unlock()
			http.Error(w, "too many requests in flight", http.StatusTooManyRequests)
			return
		}
		inFlight[ip]++
		mu.Unlock()
		defer func() {
			mu.Lock()
			if inFlight[ip]--; inFlight[ip] == 0 {
				delete(inFlight, ip)
			}
			mu.Unlock()
		}()
		next.ServeHTTP(w, r)
	})
}
