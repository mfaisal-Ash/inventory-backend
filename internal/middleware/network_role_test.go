package middleware

import (
	"testing"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

func TestNetworkAllowsRole(t *testing.T) {
	t.Cleanup(func() {
		ConfigureKaryawanNetwork(nil)
		ConfigureAdminGudangOnly(false)
	})

	const inside, outside = "192.168.1.20", "8.8.8.8"

	tests := []struct {
		name      string
		networks  []string
		adminOnly bool
		role      string
		ip        string
		want      bool
	}{
		{"tanpa pembatasan: karyawan dari luar boleh", nil, false, constant.RoleKaryawan, outside, true},
		{"karyawan di dalam gudang boleh", []string{"192.168.1.0/24"}, false, constant.RoleKaryawan, inside, true},
		{"karyawan dari luar ditolak", []string{"192.168.1.0/24"}, false, constant.RoleKaryawan, outside, false},
		{"admin dari luar boleh kalau tidak dibatasi", []string{"192.168.1.0/24"}, false, constant.RoleAdmin, outside, true},
		{"admin dari luar ditolak kalau dibatasi", []string{"192.168.1.0/24"}, true, constant.RoleAdmin, outside, false},
		{"admin di dalam gudang boleh kalau dibatasi", []string{"192.168.1.0/24"}, true, constant.RoleAdmin, inside, true},
		{"super admin tidak pernah dibatasi", []string{"192.168.1.0/24"}, true, constant.RoleSuperAdmin, outside, true},
		{"role tak dikenal fail-closed", []string{"192.168.1.0/24"}, false, "tamu", outside, false},
		{"admin dibatasi tapi daftar kosong = tanpa pembatasan", nil, true, constant.RoleAdmin, outside, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ConfigureKaryawanNetwork(tt.networks)
			ConfigureAdminGudangOnly(tt.adminOnly)
			if got := NetworkAllowsRole(tt.role, tt.ip); got != tt.want {
				t.Fatalf("NetworkAllowsRole(%q, %q) = %v, want %v", tt.role, tt.ip, got, tt.want)
			}
		})
	}
}
