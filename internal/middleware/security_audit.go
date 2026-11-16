package middleware

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// SecurityAudit mencatat indikasi probing, autentikasi gagal, dan akses terlarang.
// Catatan ini adalah sinyal keamanan, bukan bukti bahwa hacker berhasil masuk.
func SecurityAudit(logDir string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		status := c.Response().StatusCode()
		if status != fiber.StatusUnauthorized && status != fiber.StatusForbidden && status != fiber.StatusNotFound {
			return err
		}
		path := strings.ToLower(c.Path())
		suspicious := status != fiber.StatusNotFound || strings.Contains(path, "..") || strings.Contains(path, "wp-") || strings.Contains(path, ".env") || strings.Contains(path, "admin")
		if !suspicious { return err }
		event := map[string]any{"timestamp": time.Now().UTC().Format(time.RFC3339), "type": "security_event", "severity": "warning", "status": status, "method": c.Method(), "path": c.Path(), "ip": c.IP(), "user_agent": c.Get("User-Agent"), "message": "Permintaan mencurigakan atau ditolak; bukan bukti login berhasil"}
		if logDir == "" { logDir = "./log" }
		if mkErr := os.MkdirAll(logDir, 0750); mkErr == nil {
			if f, openErr := os.OpenFile(filepath.Join(logDir, "security.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); openErr == nil {
				defer f.Close(); b, _ := json.Marshal(event); _, _ = f.Write(append(b, '\n'))
			}
		}
		log.Printf("SECURITY_EVENT status=%d method=%s path=%s ip=%s", status, c.Method(), c.Path(), c.IP())
		return err
	}
}
