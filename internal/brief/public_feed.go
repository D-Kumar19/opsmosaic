package brief

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

var nonpublicFeedRanges = func() []netip.Prefix {
	var ranges []netip.Prefix
	for _, prefix := range []string{
		"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24",
		"192.88.99.0/24", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"240.0.0.0/4", "2001::/23", "2001:db8::/32", "2002::/16", "3fff::/20",
	} {
		ranges = append(ranges, netip.MustParsePrefix(prefix))
	}
	return ranges
}()

func publicFeedIP(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return false
	}
	address = address.Unmap()
	if !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() ||
		address.IsLinkLocalUnicast() {
		return false
	}
	if address.Is6() && !netip.MustParsePrefix("2000::/3").Contains(address) {
		return false
	}
	for _, prefix := range nonpublicFeedRanges {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func publicFeedURL(raw string) (*url.URL, error) {
	if len(raw) > 2048 || safeURL(raw) != raw {
		return nil, fmt.Errorf("feed URL must be a valid public HTTPS URL (maximum 2048 characters)")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || u.ForceQuery ||
		(u.Port() != "" && u.Port() != "443") {
		return nil, fmt.Errorf("feed URL requires HTTPS on port 443, without credentials, query strings or fragments")
	}
	host := strings.ToLower(u.Hostname())
	if ip := net.ParseIP(host); ip != nil {
		if !publicFeedIP(ip) {
			return nil, fmt.Errorf("feed URL must not target a private or special-use address")
		}
	} else {
		if !strings.Contains(host, ".") || strings.HasSuffix(host, ".") || len(host) > 253 {
			return nil, fmt.Errorf("feed URL requires a public fully qualified hostname")
		}
		for _, suffix := range []string{".localhost", ".local", ".internal", ".home", ".lan", ".test", ".invalid", ".onion"} {
			if strings.HasSuffix(host, suffix) {
				return nil, fmt.Errorf("feed URL must not use a local or special-use hostname")
			}
		}
		for _, label := range strings.Split(host, ".") {
			if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return nil, fmt.Errorf("feed hostname has an invalid DNS label")
			}
			for _, ch := range label {
				if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
					return nil, fmt.Errorf("feed hostname must use ASCII DNS labels")
				}
			}
		}
		// Numeric legacy IP spellings must not be interpreted as hostnames.
		last := host[strings.LastIndex(host, ".")+1:]
		if last[0] >= '0' && last[0] <= '9' {
			return nil, fmt.Errorf("feed hostname requires a nonnumeric top-level label")
		}
	}
	u.Host = host
	if strings.Contains(host, ":") {
		u.Host = "[" + host + "]"
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u, nil
}

type feedLookup func(context.Context, string) ([]net.IPAddr, error)
type feedDial func(context.Context, string, string) (net.Conn, error)

func publicFeedTransport(lookup feedLookup, dial feedDial) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Direct connections are required to enforce DNS/IP pinning; ignore proxy env.
	transport.Proxy = nil
	transport.DialTLSContext = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || port != "443" {
			return nil, fmt.Errorf("custom feeds require port 443")
		}
		var addresses []net.IPAddr
		if ip := net.ParseIP(host); ip != nil {
			addresses = []net.IPAddr{{IP: ip}}
		} else {
			addresses, err = lookup(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("custom feed DNS lookup failed: %w", err)
			}
		}
		if len(addresses) == 0 {
			return nil, fmt.Errorf("custom feed hostname has no addresses")
		}
		for _, candidate := range addresses {
			if candidate.Zone != "" || !publicFeedIP(candidate.IP) {
				return nil, fmt.Errorf("custom feed DNS includes a private or special-use address; connection blocked")
			}
		}
		var failures []error
		for _, candidate := range addresses {
			connection, err := dial(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			if err == nil {
				return connection, nil
			}
			failures = append(failures, err)
		}
		return nil, fmt.Errorf("custom feed connection failed: %w", errors.Join(failures...))
	}
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = 5 * time.Second
	return transport
}

func publicFeedClient(transport *http.Transport) *http.Client {
	return &http.Client{
		Transport: transport, Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("custom feed exceeded five redirect hops")
			}
			_, err := publicFeedURL(req.URL.String())
			return err
		},
	}
}
