//go:build linux

package proxy

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Config struct {
	ListenPort string
	IranOnly   bool
	CIDRFile   string
	DataDir    string
	APIPort    string
	JWTSecret  string
}

var (
	iranNets     []netip.Prefix
	wg0Dialer    *net.Dialer
	directDialer *net.Dialer
)

func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func LoadConfig() Config {
	return Config{
		ListenPort: EnvOr("PROXY_PORT", "1081"),
		IranOnly:   os.Getenv("IRAN_ONLY") == "1",
		CIDRFile:   EnvOr("CIDR_FILE", "/opt/wg0proxy/ir.cidr"),
		DataDir:    EnvOr("DATA_DIR", "/opt/wg0proxy"),
		APIPort:    EnvOr("API_PORT", "8080"),
		JWTSecret:  EnvOr("JWT_SECRET", "change-me-in-production"),
	}
}

func loadIranCIDRs(path string) {
	f, err := os.Open(path)
	if err != nil {
		slog.Warn("CIDR file not found, all traffic will go through wg0", "path", path)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, err := netip.ParsePrefix(line)
		if err != nil {
			continue
		}
		iranNets = append(iranNets, prefix)
	}
	slog.Info("loaded Iran CIDR blocks", "count", len(iranNets))
}

func isIranIP(addr netip.Addr) bool {
	if len(iranNets) == 0 {
		return true
	}
	for _, prefix := range iranNets {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func makeWg0Dialer() *net.Dialer {
	return &net.Dialer{
		Timeout: 10 * time.Second,
		Control: func(_, _ string, c syscall.RawConn) error {
			var sErr error
			err := c.Control(func(fd uintptr) {
				sErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, "wg0")
			})
			if err != nil {
				return err
			}
			return sErr
		},
	}
}

func readExact(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r, buf)
	return buf, err
}

func handleConn(conn net.Conn, cfg *Config, store *Store) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	clientIP := conn.RemoteAddr().String()

	// Phase 1: SOCKS5 greeting
	header, err := readExact(conn, 2)
	if err != nil || header[0] != 0x05 {
		return
	}
	nMethods := int(header[1])
	if _, err := readExact(conn, nMethods); err != nil {
		return
	}
	// Select username/password auth (0x02)
	conn.Write([]byte{0x05, 0x02})

	// Phase 2: Username/password auth (RFC 1929)
	authVer, err := readExact(conn, 1)
	if err != nil || authVer[0] != 0x01 {
		conn.Write([]byte{0x01, 0x01})
		return
	}
	ulenBuf, err := readExact(conn, 1)
	if err != nil {
		return
	}
	ulen := int(ulenBuf[0])
	uname, err := readExact(conn, ulen)
	if err != nil {
		return
	}
	plenBuf, err := readExact(conn, 1)
	if err != nil {
		return
	}
	plen := int(plenBuf[0])
	passwd, err := readExact(conn, plen)
	if err != nil {
		return
	}

	username := string(uname)
	if !store.Authenticate(username, string(passwd)) {
		conn.Write([]byte{0x01, 0x01})
		slog.Warn("auth failed", "client", clientIP, "user", username)
		return
	}
	conn.Write([]byte{0x01, 0x00})

	// Phase 3: Connect request
	reqHeader, err := readExact(conn, 4)
	if err != nil || reqHeader[0] != 0x05 || reqHeader[1] != 0x01 {
		return
	}

	atype := reqHeader[3]
	var target string
	var port uint16

	switch atype {
	case 0x01: // IPv4
		ipBytes, err := readExact(conn, 4)
		if err != nil {
			return
		}
		target = net.IP(ipBytes).String()
		portBytes, err := readExact(conn, 2)
		if err != nil {
			return
		}
		port = binary.BigEndian.Uint16(portBytes)

	case 0x03: // Domain
		dlenBuf, err := readExact(conn, 1)
		if err != nil {
			return
		}
		dlen := int(dlenBuf[0])
		domain, err := readExact(conn, dlen)
		if err != nil {
			return
		}
		target = string(domain)
		portBytes, err := readExact(conn, 2)
		if err != nil {
			return
		}
		port = binary.BigEndian.Uint16(portBytes)

	case 0x04: // IPv6
		ipBytes, err := readExact(conn, 16)
		if err != nil {
			return
		}
		target = net.IP(ipBytes).String()
		portBytes, err := readExact(conn, 2)
		if err != nil {
			return
		}
		port = binary.BigEndian.Uint16(portBytes)

	default:
		return
	}

	// Resolve DNS to get IP for CIDR check
	var resolvedIP netip.Addr
	ips, err := net.DefaultResolver.LookupNetIP(context.Background(), "ip4", target)
	if err == nil && len(ips) > 0 {
		resolvedIP = ips[0]
	} else {
		resolvedIP, _ = netip.ParseAddr(target)
	}

	// Routing decision
	dialer := wg0Dialer
	route := "wg0"
	if cfg.IranOnly && resolvedIP.IsValid() {
		if !isIranIP(resolvedIP) {
			dialer = directDialer
			route = "direct"
		}
	}

	resolvedStr := resolvedIP.String()
	if !resolvedIP.IsValid() {
		resolvedStr = "unresolved"
	}
	slog.Info("connect", "client", clientIP, "user", username, "target", fmt.Sprintf("%s:%d", target, port), "resolved", resolvedStr, "route", route)

	// Dial target using resolved IP to avoid double DNS
	dialAddr := target
	if resolvedIP.IsValid() {
		dialAddr = resolvedIP.String()
	}

	remote, err := dialer.DialContext(context.Background(), "tcp", net.JoinHostPort(dialAddr, fmt.Sprintf("%d", port)))
	if err != nil {
		slog.Error("dial failed", "target", fmt.Sprintf("%s:%d", target, port), "route", route, "err", err)
		conn.Write([]byte{0x05, 0x05, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		return
	}
	defer remote.Close()

	// Send SOCKS5 success reply
	conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0})

	// Clear deadline for relay phase
	conn.SetDeadline(time.Time{})
	ul, dl := relayWithTraffic(conn, remote)
	store.AddTraffic(username, ul, dl)
}

// FormatBytes returns a human-readable byte size.
func FormatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func RunServe(cfg *Config, store *Store) {
	if cfg.IranOnly {
		loadIranCIDRs(cfg.CIDRFile)
	}

	wg0Dialer = makeWg0Dialer()
	directDialer = &net.Dialer{Timeout: 10 * time.Second}

	if !store.HasUsers() {
		slog.Error("no users configured, add users first: wg0proxy user add <username> <password>")
		os.Exit(1)
	}

	mode := "all traffic via wg0"
	if cfg.IranOnly {
		mode = "Iran-only via wg0"
	}

	ln, err := net.Listen("tcp", ":"+cfg.ListenPort)
	if err != nil {
		slog.Error("listen failed", "err", err)
		os.Exit(1)
	}

	slog.Info("SOCKS5 proxy started", "port", cfg.ListenPort, "mode", mode)

	// Start API server
	go runAPI(cfg, store)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Periodic traffic flush every 30 seconds
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := store.Flush(); err != nil {
					slog.Error("traffic flush failed", "err", err)
				}
			case <-ctx.Done():
				store.Flush()
				return
			}
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("shutting down")
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("accept error", "err", err)
			continue
		}
		go handleConn(conn, cfg, store)
	}
}
