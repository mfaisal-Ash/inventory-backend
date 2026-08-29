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
		header := c.Get("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			return utils.Fail(c, fiber.StatusUnauthorized, "token tidak ditemukan", nil)
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtSvc.ParseAccessToken(tokenStr)
		if err != nil {
			return utils.Fail(c, fiber.StatusUnauthorized, "token tidak valid atau kedaluwarsa", nil)
		}

		revoked, revokedByUsername, revokeErr := jwtSvc.CheckSession(claims.SessionID)
		if revokeErr == nil && revoked {
			if revokedByUsername != "" {
				return utils.Fail(c, fiber.StatusUnauthorized, ForcedLogoutMessagePrefix+revokedByUsername, nil)
			}
			return utils.Fail(c, fiber.StatusUnauthorized, "sesi ini sudah dicabut, silakan login ulang", nil)
		}

		c.Locals(constant.CtxUserID, claims.UserID)
		c.Locals(constant.CtxRoleID, claims.RoleID)
		c.Locals(constant.CtxRoleName, claims.RoleName)
		c.Locals(constant.CtxSessionID, claims.SessionID)
		return c.Next()
	}
}
