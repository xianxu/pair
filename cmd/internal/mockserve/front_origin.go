package mockserve

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func parseRealOrigin(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid real app origin: %w", err)
	}
	if u.Scheme != "http" || u.User != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return nil, fmt.Errorf("real app must be an HTTP loopback origin with an explicit port and no path, credentials, query or fragment")
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return nil, fmt.Errorf("real app origin must be loopback")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("real app origin needs a port from 1 to 65535")
	}
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u, nil
}

// loopbackEndpoint uses the same pinned address for dialing and identity checks.
// A scheme-less authority in a redirect inherits the front's HTTP scheme.
func loopbackEndpoint(u *url.URL) (net.TCPAddr, bool) {
	if u == nil || u.User != nil || u.Host == "" || u.Scheme != "" && !strings.EqualFold(u.Scheme, "http") {
		return net.TCPAddr{}, false
	}
	host := u.Hostname()
	var ip net.IP
	if strings.EqualFold(host, "localhost") {
		ip = net.ParseIP("127.0.0.1")
	} else {
		ip = net.ParseIP(host)
	}
	if ip == nil || !ip.IsLoopback() {
		return net.TCPAddr{}, false
	}
	portText := u.Port()
	if portText == "" {
		portText = "80"
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return net.TCPAddr{}, false
	}
	return net.TCPAddr{IP: ip, Port: port}, true
}

func targetsListener(origin *url.URL, listener *net.TCPAddr) bool {
	endpoint, ok := loopbackEndpoint(origin)
	return ok && endpoint.Port == listener.Port && (listener.IP.IsUnspecified() || endpoint.IP.Equal(listener.IP))
}

func rewriteRealLocation(raw string, origin *url.URL) string {
	location, err := url.Parse(raw)
	if err != nil || location.Host == "" {
		return raw
	}
	target, ok := loopbackEndpoint(location)
	upstream, valid := loopbackEndpoint(origin)
	if !ok || !valid || target.Port != upstream.Port || !target.IP.Equal(upstream.IP) {
		return raw
	}
	location.Scheme, location.Host = "", ""
	if location.Path == "" {
		location.Path = "/"
		location.RawPath = ""
	}
	// A path beginning // must remain a path, never a network-path authority.
	// Resolving the dot prefix keeps the original path and the front's host.
	if strings.HasPrefix(location.Path, "//") {
		location.Path = "/." + location.Path
		if location.RawPath != "" {
			location.RawPath = "/." + location.RawPath
		}
	}
	return location.String()
}
