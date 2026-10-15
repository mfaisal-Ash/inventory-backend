package middleware

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/activitylog"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
)

// ActivityRecorder dipenuhi oleh repository activity_log.
type ActivityRecorder interface {
	Create(l *model.ActivityLog) error
}

const apiPrefix = "/stockrsd"

// ActivityLogger mencatat setiap request TULIS (POST/PUT/PATCH/DELETE) milik
// user yang sudah login ke tabel activity_logs, termasuk yang gagal. Login
// dan logout ikut tercatat. Pencatatan dilakukan setelah handler selesai
// (supaya status & user dari JWT sudah diketahui) dan penyimpanan ke DB
// berjalan di goroutine terpisah agar tidak menambah latensi request.
//
// Pasang SEBELUM route didaftarkan: api.Use(middleware.ActivityLogger(repo)).
func ActivityLogger(rec ActivityRecorder) fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		if method == fiber.MethodGet || method == fiber.MethodHead || method == fiber.MethodOptions {
			return c.Next()
		}

		rel := strings.TrimSuffix(strings.TrimPrefix(c.Path(), apiPrefix), "/")
		if activitylog.ShouldSkip(rel) {
			return c.Next()
		}

		// Salin body & string dari ctx SEBELUM Next(): Fiber memakai ulang
		// buffer request setelah handler selesai.
		// Body hanya dibaca untuk jalur /auth (mengambil username percobaan login);
		// isi request lain TIDAK disimpan di riwayat.
		var body []byte
		if strings.HasPrefix(rel, "/auth/") && strings.Contains(c.Get(fiber.HeaderContentType), "application/json") {
			body = append([]byte(nil), c.Body()...)
		}
		ip := strings.Clone(c.IP())
		ua := activitylog.Truncate(strings.Clone(c.Get(fiber.HeaderUserAgent)), 250)
		fullPath := strings.Clone(c.Path())
		// Import massal: tiap baris dikirim sebagai request tambah biasa yang
		// ditandai header ini. Tetap dicatat satu per satu (header ini hanya
		// pelabel, bukan cara menghindari audit) tetapi diberi aksi "import".
		source := strings.Clone(c.Get("X-Activity-Source"))

		handlerErr := c.Next()

		status := c.Response().StatusCode()
		if handlerErr != nil {
			status = fiber.StatusInternalServerError
			if fe, ok := handlerErr.(*fiber.Error); ok {
				status = fe.Code
			}
		}
		success := status < 400

		userID, _ := c.Locals(constant.CtxUserID).(uint)
		roleName, _ := c.Locals(constant.CtxRoleName).(string)

		info := activitylog.Classify(method, rel)

		// Aksi tanpa user login hanya dicatat untuk jalur autentikasi.
		username := ""
		if userID == 0 {
			switch rel {
			case "/auth/login":
				if success {
					// Login 2FA: sesi baru terbit di /auth/verify-otp.
					return handlerErr
				}
				username = activitylog.UsernameFromBody(body)
			case "/auth/register", "/auth/verify-otp", "/auth/password/forgot":
				username = activitylog.UsernameFromBody(body)
			default:
				return handlerErr
			}
		}

		if source == "import" && info.Action == activitylog.ActionCreate {
			info.Action = activitylog.ActionImport
		}

		errMsg := ""
		if !success {
			errMsg = extractMessage(c.Response().Body())
			if errMsg == "" && handlerErr != nil {
				errMsg = handlerErr.Error()
			}
			errMsg = activitylog.Truncate(errMsg, 200)
		}

		entry := &model.ActivityLog{
			Username:    username,
			RoleName:    strings.Clone(roleName),
			Action:      info.Action,
			Module:      info.Module,
			ModuleLabel: activitylog.ModuleLabel(info.Module),
			EntityID:    info.EntityID,
			Description: activitylog.Describe(info.Action, info.Module, info.EntityID, success, errMsg),
			Method:      method,
			Path:        activitylog.Truncate(fullPath, 250),
			StatusCode:  status,
			Success:     success,
			IPAddress:   ip,
			UserAgent:   ua,
			Detail:      errMsg,
			CreatedAt:   time.Now(),
		}
		if userID != 0 {
			uid := userID
			entry.UserID = &uid
		}

		go func(e *model.ActivityLog) {
			if err := rec.Create(e); err != nil {
				log.Printf("activity-log: gagal menyimpan riwayat (%s %s): %v", e.Method, e.Path, err)
			}
		}(entry)

		return handlerErr
	}
}

// extractMessage mengambil field "message" dari envelope respons error.
func extractMessage(body []byte) string {
	var env struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return ""
	}
	return env.Message
}
