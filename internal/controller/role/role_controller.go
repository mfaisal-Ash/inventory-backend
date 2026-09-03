package role

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

func (h *Controller) List(c *fiber.Ctx) error {
	roles, err := h.repo.FindAll()
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil daftar role", nil)
	}
	return utils.OK(c, "daftar role berhasil diambil", roles)
}

func (h *Controller) Create(c *fiber.Ctx) error {
	var req CreateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "payload tidak valid", nil)
	}
	if errs := utils.Validate(req); errs != nil {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "validasi gagal", errs)
	}

	if _, err := h.repo.FindByName(req.Name); err == nil {
		return utils.Fail(c, fiber.StatusConflict, "nama role sudah digunakan", nil)
	}

	roleModel := &model.Role{Name: req.Name, Description: req.Description}
	if err := h.repo.Create(roleModel); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat role", nil)
	}
	return utils.Created(c, "role berhasil dibuat", roleModel)
}

func (h *Controller) GetPermissionMatrix(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id role tidak valid", nil)
	}

	// Setiap user (bukan cuma super admin) perlu bisa baca matrix akses
	// ROLE-NYA SENDIRI — usePermissions.ts di frontend memanggil endpoint
	// ini untuk semua role non-super-admin supaya tahu menu/tombol apa saja
	// yang boleh diakses. Kalau ini diblokir total ke super_admin saja,
	// SEMUA user admin/karyawan gagal memuat izinnya sendiri dan otomatis
	// dianggap tidak punya akses ke mana pun (redirect 403 di semua
	// halaman). Jadi izinkan: super_admin (lihat matrix role apa saja),
	// atau user biasa yang memang meminta matrix ROLE-NYA SENDIRI. Melihat
	// matrix milik role LAIN tetap diblokir di sini (bukan cuma di
	// RegisterRoutes) supaya information-disclosure fix sebelumnya tetap utuh.
	roleName, _ := c.Locals(constant.CtxRoleName).(string)
	roleID, _ := c.Locals(constant.CtxRoleID).(uint)
	if roleName != constant.RoleSuperAdmin && roleID != uint(id) {
		return utils.Fail(c, fiber.StatusForbidden, "tidak diizinkan melihat matrix akses role lain", nil)
	}

	matrix, err := h.repo.GetMatrix(uint(id))
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengambil matrix akses", nil)
	}
	return utils.OK(c, "matrix akses berhasil diambil", fiber.Map{"role_id": id, "items": matrix})
}

func (h *Controller) UpdatePermissionMatrix(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id role tidak valid", nil)
	}

	var req UpdatePermissionMatrixRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "payload tidak valid", nil)
	}
	if errs := utils.Validate(req); errs != nil {
		return utils.Fail(c, fiber.StatusUnprocessableEntity, "validasi gagal", errs)
	}

	var permissionIDs []uint
	for _, item := range req.Items {
		actions := []struct {
			enabled bool
			action  string
		}{
			{item.View, constant.ActionView},
			{item.Tambah, constant.ActionTambah},
			{item.Edit, constant.ActionEdit},
			{item.ApprovalReject, constant.ActionApprovalReject},
			{item.Print, constant.ActionPrint},
			{item.AssignDelegasi, constant.ActionAssignDelegasi},
		}

		for _, a := range actions {
			if !a.enabled {
				continue
			}
			p, err := h.repo.FindOrCreatePermission(item.Module, a.action)
			if err != nil {
				return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan matrix akses", nil)
			}
			permissionIDs = append(permissionIDs, p.ID)
		}
	}

	if err := h.repo.ReplaceRolePermissions(uint(id), permissionIDs); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan matrix akses", nil)
	}
	return utils.OK(c, "matrix akses berhasil diperbarui", nil)
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/roles", middleware.JWTAuth(h.jwtSvc))
	g.Get("/", h.List)
	g.Post("/", middleware.RequireRole(constant.RoleSuperAdmin), h.Create)
	// GetPermissionMatrix TIDAK dibatasi RequireRole(super_admin) di sini —
	// endpoint ini juga dipakai tiap user non-super-admin buat baca matrix
	// akses ROLE-NYA SENDIRI (lihat usePermissions.ts). Pengecekan "boleh
	// lihat role sendiri, atau super_admin buat role manapun" dilakukan di
	// dalam handler (butuh bandingkan :id dengan roleID user, bukan cuma
	// nama role) — lihat komentar di GetPermissionMatrix.
	g.Get("/:id/permissions", h.GetPermissionMatrix)
	g.Put("/:id/permissions", middleware.RequireRole(constant.RoleSuperAdmin), h.UpdatePermissionMatrix)
}
