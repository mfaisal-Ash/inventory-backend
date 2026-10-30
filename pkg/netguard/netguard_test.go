package netguard

import "testing"

func TestGuardAllows(t *testing.T) {
	tests := []struct {
		name    string
		entries []string
		ip      string
		want    bool
	}{
		{"tanpa pembatasan", nil, "8.8.8.8", true},
		{"entri kosong dianggap tanpa pembatasan", []string{"", "  "}, "8.8.8.8", true},
		{"cidr cocok", []string{"192.168.1.0/24"}, "192.168.1.77", true},
		{"cidr tidak cocok", []string{"192.168.1.0/24"}, "192.168.2.1", false},
		{"ip tunggal cocok", []string{"203.0.113.10"}, "203.0.113.10", true},
		{"ip tunggal tidak cocok", []string{"203.0.113.10"}, "203.0.113.11", false},
		{"beberapa entri", []string{"10.0.0.0/8", "192.168.1.0/24"}, "10.5.5.5", true},
		{"ipv6", []string{"fd00::/8"}, "fd12::1", true},
		{"ip tidak valid ditolak", []string{"192.168.1.0/24"}, "bukan-ip", false},
		{"semua entri tidak valid tetap menolak", []string{"salah"}, "192.168.1.5", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, _ := New(tt.entries)
			if got := g.Allows(tt.ip); got != tt.want {
				t.Fatalf("Allows(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestNewReportsInvalid(t *testing.T) {
	_, invalid := New([]string{"192.168.1.0/24", "salah", "300.1.1.1"})
	if len(invalid) != 2 {
		t.Fatalf("invalid = %v, want 2 entri", invalid)
	}
}

func TestNilGuardAllowsAll(t *testing.T) {
	var g *Guard
	if !g.Allows("1.2.3.4") {
		t.Fatal("Guard nil harus mengizinkan semua")
	}
}
