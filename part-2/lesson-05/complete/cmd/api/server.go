package main

import (
	"log/slog"
	"net/http"
	"time"
)

// maxBodyBytes is the largest request body we read: 1 MB is plenty for a
// task or an invoice, and it caps what one request can cost us in memory.
const maxBodyBytes = 1 << 20

// limits are the server's timeouts. Each one closes a door that a slow or
// malicious client could otherwise keep open forever.
type limits struct {
	ReadHeader time.Duration // the request line and headers must arrive within this
	Read       time.Duration // the whole request, body included
	Write      time.Duration // from the end of the headers to the end of our response
	Idle       time.Duration // a keep-alive connection waiting for its next request
	MaxHeader  int           // bytes of request line + headers
}

// defaultLimits are generous for real clients and short for attackers.
// Write must stay above our slowest handler (/debug/slow waits up to 30 s).
var defaultLimits = limits{
	ReadHeader: 5 * time.Second,
	Read:       15 * time.Second,
	Write:      40 * time.Second,
	Idle:       60 * time.Second,
	MaxHeader:  16 << 10, // 16 KB, the default is 1 MB
}

// newServer is http.Server with every limit set. The zero value of each of
// these fields means "no limit", and that is what Slowloris relies on.
func newServer(addr string, h http.Handler, l limits) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: l.ReadHeader,
		ReadTimeout:       l.Read,
		WriteTimeout:      l.Write,
		IdleTimeout:       l.Idle,
		MaxHeaderBytes:    l.MaxHeader,
		// net/http's own messages (like "Accept error") go to our JSON log too.
		ErrorLog: slog.NewLogLogger(slog.Default().Handler(), slog.LevelWarn),
	}
}
