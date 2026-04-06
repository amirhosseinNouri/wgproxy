//go:build linux

package proxy

import (
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// countingReader wraps an io.Reader and counts bytes read.
type countingReader struct {
	r     io.Reader
	bytes atomic.Int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.bytes.Add(int64(n))
	return n, err
}

// countingWriter wraps an io.Writer and counts bytes written.
type countingWriter struct {
	w     io.Writer
	bytes atomic.Int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.bytes.Add(int64(n))
	return n, err
}

// relayWithTraffic relays data between client and remote while counting
// bytes transferred. Returns (upload, download) byte counts.
// Upload = client -> remote, Download = remote -> client.
func relayWithTraffic(client, remote net.Conn) (upload, download int64) {
	var wg sync.WaitGroup
	wg.Add(2)

	ul := &countingReader{r: client}
	dl := &countingReader{r: remote}

	cp := func(dst net.Conn, src *countingReader) {
		defer wg.Done()
		io.Copy(dst, src)
		if tc, ok := dst.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}

	go cp(remote, ul)
	go cp(client, dl)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(60 * time.Second):
	}

	return ul.bytes.Load(), dl.bytes.Load()
}
