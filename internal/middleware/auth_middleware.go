package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const ForcedLogoutMessagePrefix = "akun ini di keluarkan secara paksa oleh "

func JWTAuth(jwtSvc *utils.JWTService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := strings.TrimSpace(c.Get("Authorization"))
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return utils.Fail(c, fiber.StatusUnauthorized, "token tidak ditemukan", nil)
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtSvc.ParseAccessToken(tokenStr)
		if err != nil {
			return utils.Fail(c, fiber.StatusUnauthorized, "token tidak valid atau kedaluwarsa", nil)
		}

		revoked, revokedByUsername, revokeErr := jwtSvc.CheckSession(claims.SessionID)
		if revokeErr != nil {
			return utils.Fail(c, fiber.StatusServiceUnavailable, "status sesi tidak dapat diverifikasi", nil)
		}
		if revoked {
			if revokedByUsername != "" {
				return utils.Fail(c, fiber.StatusUnauthorized, ForcedLogoutMessagePrefix+revokedByUsername, nil)
			}
			return utils.Fail(c, fiber.StatusUnauthorized, "sesi ini sudah dicabut, silakan login ulang", nil)
		}

		// Akun admin & karyawan hanya boleh dipakai dari sekitar titik koordinat
		// gudang (dan jaringan gudang, kalau diatur). Dicek di setiap request, bukan
		// cuma saat login, supaya token yang dibawa keluar gudang tidak berguna.
		// Super admin tidak dibatasi.
		if ok, msg := AccessAllowsRole(c, claims.RoleName); !ok {
			return utils.Fail(c, fiber.StatusForbidden, msg, nil)
		}

		c.Locals(constant.CtxUserID, claims.UserID)
		c.Locals(constant.CtxRoleID, claims.RoleID)
		c.Locals(constant.CtxRoleName, claims.RoleName)
		c.Locals(constant.CtxSessionID, claims.SessionID)
		return c.Next()
	}
}
