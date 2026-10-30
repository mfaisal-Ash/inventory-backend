package users

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	cons "github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/netguard"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

// manageableTarget memuat akun target dari :id dan memastikan targetnya admin
// atau karyawan. Akun super admin sengaja tidak bisa dikendalikan lewat fitur
// ini (termasuk milik sendiri) supaya tidak ada yang mengunci super admin.
// Kalau gagal, respons error sudah ditulis dan ok = false.
func (h *Controller) manageableTarget(c *fiber.Ctx) (*model.User, bool) {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		_ = utils.Fail(c, fiber.StatusBadRequest, cons.ErrIDInvalid, nil)
		return nil, false
	}
	u, err := h.userRepo.FindByID(uint(id))
	if err != nil {
		_ = utils.Fail(c, fiber.StatusNotFound, cons.ErrUsersUserNotFound, nil)
		return nil, false
	}
	targetRole, err := h.roleRepo.FindByID(u.RoleID)
	if err != nil || targetRole.Name == cons.RoleSuperAdmin {
		_ = utils.Fail(c, fiber.StatusForbidden, "akun super admin tidak bisa dikendalikan lewat fitur ini", nil)
		return nil, false
	}
	return u, true
}

// ResetPassword: super admin mengatur ulang password akun admin/karyawan, lalu
// semua sesi akun itu dicabut dan kunci login (percobaan gagal) dibersihkan.
func (h *Controller) ResetPassword(c *fiber.Ctx) error {
	var req ResetPasswordRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}
	u, ok := h.manageableTarget(c)
	if !ok {
		return nil
	}

	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, cons.ErrHashPasswordFail, nil)
	}
	u.PasswordHash = hashed
	if err := h.userRepo.Update(u); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, cons.ErrGagalMengubahPassword, nil)
	}
	_ = h.userRepo.ResetFailedLogin(u.ID)
	_ = h.authRepo.RevokeAllUserTokens(u.ID)
	return utils.OK(c, "password berhasil direset, semua sesi akun tersebut otomatis ter-logout", h.toResponseSingle(u))
}

// ForceLogout: super admin mencabut semua sesi aktif akun admin/karyawan.
func (h *Controller) ForceLogout(c *fiber.Ctx) error {
	u, ok := h.manageableTarget(c)
	if !ok {
		return nil
	}
	if err := h.authRepo.RevokeAllUserTokens(u.ID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mencabut sesi", nil)
	}
	return utils.OK(c, "semua sesi akun tersebut dicabut dan otomatis ter-logout", nil)
}

func (h *Controller) accessPolicyResponse(c *fiber.Ctx) AccessPolicyResponse {
	ip := c.IP()
	res := AccessPolicyResponse{
		KaryawanNetworks: middleware.KaryawanNetworkEntries(),
		Restricted:       middleware.KaryawanNetworkEnabled(),
		AdminGudangOnly:  middleware.AdminGudangOnlyEnabled(),
		ClientIP:         ip,
		ClientIPAllowed:  middleware.KaryawanNetworkAllows(ip),
		Source:           "env",
	}
	if h.accessPolicyRepo != nil {
		if p, err := h.accessPolicyRepo.Get(); err == nil && p != nil {
			updatedAt := p.UpdatedAt
			res.Source = "database"
			res.UpdatedAt = &updatedAt
		}
	}
	return res
}

// GetAccessPolicy: kebijakan jaringan gudang untuk akun karyawan yang sedang
// berlaku, plus IP pemanggil (supaya super admin bisa menambahkan IP-nya).
func (h *Controller) GetAccessPolicy(c *fiber.Ctx) error {
	return utils.OK(c, "kebijakan akses karyawan berhasil dimuat", h.accessPolicyResponse(c))
}

// UpdateAccessPolicy menyimpan daftar jaringan gudang dan langsung
// memberlakukannya tanpa restart. Daftar kosong mematikan pembatasan.
// Karyawan selalu mengikuti daftar ini; admin ikut hanya kalau admin_gudang_only
// dinyalakan. Super admin tidak pernah dibatasi, jadi tidak ada risiko terkunci.
func (h *Controller) UpdateAccessPolicy(c *fiber.Ctx) error {
	var req UpdateAccessPolicyRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, cons.ErrPayloadInvalid, nil)
	}
	if h.accessPolicyRepo == nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "penyimpanan kebijakan akses belum tersedia", nil)
	}

	entries := make([]string, 0, len(req.KaryawanNetworks))
	for _, raw := range req.KaryawanNetworks {
		entries = append(entries, netguard.Split(raw)...)
	}
	if _, invalid := netguard.New(entries); len(invalid) > 0 {
		return utils.Fail(c, fiber.StatusUnprocessableEntity,
			"entri jaringan tidak valid (pakai IP seperti 203.0.113.10 atau CIDR seperti 192.168.1.0/24): "+strings.Join(invalid, ", "), nil)
	}

	adminOnly := middleware.AdminGudangOnlyEnabled()
	if req.AdminGudangOnly != nil {
		adminOnly = *req.AdminGudangOnly
	}
	if adminOnly && len(entries) == 0 {
		return utils.Fail(c, fiber.StatusUnprocessableEntity,
			"isi dulu jaringan gudang (IP/CIDR) sebelum membatasi akun admin", nil)
	}

	userID, _ := c.Locals(cons.CtxUserID).(uint)
	if _, err := h.accessPolicyRepo.SaveAccessPolicy(strings.Join(entries, ","), adminOnly, userID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan kebijakan akses", nil)
	}
	middleware.ConfigureKaryawanNetwork(entries)
	middleware.ConfigureAdminGudangOnly(adminOnly)
	return utils.OK(c, "kebijakan akses karyawan berhasil disimpan", h.accessPolicyResponse(c))
}
