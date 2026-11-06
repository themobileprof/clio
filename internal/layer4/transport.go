package layer4

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

const maxResponseBytes = 4096

// dnsFallback helps Termux when system DNS fails for the registry host.
var dnsFallback = map[string]string{
	"clio.themobileprof.com": "157.230.148.144",
}

// NewHTTPClient is the shared client for registry and GitHub calls.
// It falls back when Termux DNS fails, and it trusts ISRG Root X1 so a
// phone with an empty system store can still verify the registry.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return newHTTPClient(timeout)
}

func newHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := dialer.DialContext(ctx, network, addr)
			if err == nil {
				return conn, nil
			}
			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, err
			}
			if ip, ok := dnsFallback[host]; ok {
				return dialer.DialContext(ctx, network, net.JoinHostPort(ip, port))
			}
			return nil, err
		},
		MaxIdleConns:        2,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 8 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    rootCAs(),
		},
	}
	return &http.Client{Timeout: timeout, Transport: transport}
}
