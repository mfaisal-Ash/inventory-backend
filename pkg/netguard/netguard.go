package netguard

import (
	"net"
	"strings"
)

type Guard struct {
	enabled bool
	nets    []*net.IPNet
}

func New(entries []string) (*Guard, []string) {
	g := &Guard{}
	var invalid []string
	for _, raw := range entries {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		g.enabled = true
		if _, n, err := net.ParseCIDR(entry); err == nil {
			g.nets = append(g.nets, n)
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			invalid = append(invalid, entry)
			continue
		}
		bits := 128
		if ip.To4() != nil {
			ip = ip.To4()
			bits = 32
		}
		g.nets = append(g.nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
	}
	return g, invalid
}

func Split(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\r' || r == '\t'
	})
}

func (g *Guard) Enabled() bool { return g != nil && g.enabled }

func (g *Guard) Allows(ip string) bool {
	if !g.Enabled() {
		return true
	}
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return false
	}
	for _, n := range g.nets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}
