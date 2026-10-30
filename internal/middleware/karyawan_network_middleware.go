package middleware

import (
	"log"
	"strings"
	"sync/atomic"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/netguard"
)

// KaryawanNetworkMessage: pesan penolakan untuk akun karyawan yang mengakses
// dari luar jaringan gudang.
const KaryawanNetworkMessage = "akun karyawan hanya bisa diakses dari dalam jaringan gudang"

// AdminNetworkMessage: pesan penolakan untuk akun admin yang mengakses dari luar
// jaringan gudang (hanya berlaku kalau super admin menyalakan pembatasan admin).
const AdminNetworkMessage = "akun admin hanya bisa diakses dari dalam jaringan gudang"

var (
	karyawanGuard   atomic.Pointer[netguard.Guard]
	karyawanEntries atomic.Pointer[[]string]
	// adminGudangOnly: true = akun admin ikut dibatasi ke daftar jaringan gudang.
	adminGudangOnly atomic.Bool
)

// ConfigureKaryawanNetwork memasang daftar jaringan gudang. Dipanggil saat router
// dibuat (nilai dari KARYAWAN_ALLOWED_NETWORKS atau dari kebijakan yang tersimpan
// di database) dan setiap kali super admin mengubahnya. Daftar kosong = tidak ada
// pembatasan.
func ConfigureKaryawanNetwork(entries []string) {
	clean := make([]string, 0, len(entries))
	for _, e := range entries {
		if t := strings.TrimSpace(e); t != "" {
			clean = append(clean, t)
		}
	}
	g, invalid := netguard.New(clean)
	for _, e := range invalid {
		log.Printf("netguard: entri jaringan karyawan '%s' tidak valid dan diabaikan", e)
	}
	if g.Enabled() {
		log.Printf("netguard: akun karyawan dibatasi ke jaringan gudang (%d entri valid)", len(clean)-len(invalid))
	}
	karyawanGuard.Store(g)
	karyawanEntries.Store(&clean)
}

// KaryawanNetworkAllows true kalau ip boleh dipakai akun karyawan.
func KaryawanNetworkAllows(ip string) bool {
	return karyawanGuard.Load().Allows(ip)
}

// KaryawanNetworkEntries mengembalikan salinan daftar jaringan yang sedang berlaku.
func KaryawanNetworkEntries() []string {
	p := karyawanEntries.Load()
	if p == nil {
		return []string{}
	}
	return append([]string{}, (*p)...)
}

// ConfigureAdminGudangOnly menyalakan/mematikan pembatasan jaringan untuk akun
// admin. Memakai daftar jaringan yang sama dengan karyawan. Super admin TIDAK
// pernah dibatasi, jadi selalu ada akun yang bisa memperbaiki kebijakan ini.
func ConfigureAdminGudangOnly(on bool) {
	adminGudangOnly.Store(on)
	if on {
		log.Printf("netguard: akun admin ikut dibatasi ke jaringan gudang")
	}
}

// AdminGudangOnlyEnabled true kalau akun admin ikut dibatasi ke jaringan gudang.
func AdminGudangOnlyEnabled() bool { return adminGudangOnly.Load() }

// NetworkAllowsRole true kalau akun dengan role tersebut boleh dipakai dari ip.
//   - super_admin: tidak pernah dibatasi (supaya tidak ada skenario terkunci total).
//   - admin: dibatasi hanya kalau pembatasan admin dinyalakan.
//   - karyawan dan role lain yang tidak dikenal: mengikuti daftar jaringan gudang
//     (fail-closed untuk role tak dikenal).
// Daftar jaringan kosong berarti tidak ada pembatasan sama sekali.
func NetworkAllowsRole(role, ip string) bool {
	switch role {
	case constant.RoleSuperAdmin:
		return true
	case constant.RoleAdmin:
		if !adminGudangOnly.Load() {
			return true
		}
		return KaryawanNetworkAllows(ip)
	default:
		return KaryawanNetworkAllows(ip)
	}
}

// NetworkMessageForRole: pesan penolakan yang sesuai role.
func NetworkMessageForRole(role string) string {
	if role == constant.RoleAdmin {
		return AdminNetworkMessage
	}
	return KaryawanNetworkMessage
}

// KaryawanNetworkEnabled true kalau pembatasan jaringan karyawan sedang aktif.
func KaryawanNetworkEnabled() bool {
	return karyawanGuard.Load().Enabled()
}
