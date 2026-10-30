package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/mfaisal-Ash/inventory-backend/internal/middleware"
	"github.com/mfaisal-Ash/inventory-backend/internal/model"
	"github.com/mfaisal-Ash/inventory-backend/pkg/constant"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geoip"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
)

const (
	maxFailedLoginAttempts = 10
	lockDuration           = 15 * time.Minute

	accountInactiveMessage     = "Akun kamu terkunci, hubungi admin supaya bisa dibuka lagi akunnya."
	otpCodeReusedMessage       = "kode sudah tidak bisa digunakan, coba gunakan kode yang baru."
	failedLoginProccessMessage = "gagal memproses login."
)

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (h *Controller) buildSessionInfo(c *fiber.Ctx) (device utils.DeviceInfo, ip, location string) {
	device = utils.ParseUserAgent(c.Get("User-Agent"))
	ip = c.IP()

	if ip == "" {
		ip = c.Context().RemoteIP().String()
	}

	location, err := h.geoipSvc.Lookup(c.Context(), ip)
	if err != nil {
		log.Printf("auth: lookup geoip untuk IP %s gagal: %v", ip, err)
		location = ""
	}

	if location == "" || location == "-" {
		if tz := c.Get("X-Timezone"); tz != "" {
			location = geoip.LocationFromTimezone(tz)
		}
	}
	return device, ip, location
}

// accessDeniedError: akun (admin/karyawan) mencoba mendapat sesi dari luar
// area gudang (lokasi koordinat atau jaringan). Pesannya dikirim apa adanya ke
// pengguna supaya jelas kenapa ditolak.
type accessDeniedError struct{ msg string }

func (e *accessDeniedError) Error() string { return e.msg }

// failIssueTokens menerjemahkan kegagalan issueTokens menjadi respons HTTP:
// 403 untuk akses dari luar gudang, selain itu 500 dengan pesan fallback.
func failIssueTokens(c *fiber.Ctx, err error, fallback string) error {
	var denied *accessDeniedError
	if errors.As(err, &denied) {
		return utils.Fail(c, fiber.StatusForbidden, denied.msg, nil)
	}
	return utils.Fail(c, fiber.StatusInternalServerError, fallback, nil)
}

func (h *Controller) issueTokens(c *fiber.Ctx, u *model.User) (*LoginResponse, error) {
	r, err := h.roleRepo.FindByID(u.RoleID)
	if err != nil {
		return nil, err
	}
	// Satu-satunya pintu penerbitan token (login, OTP, 2FA, refresh, register):
	// cek lokasi/jaringan gudang di sini supaya tidak ada jalur yang terlewat.
	if ok, msg := middleware.AccessAllowsRole(c, r.Name); !ok {
		log.Printf("auth: sesi '%s' (%s) ditolak dari IP %s: %s", u.Username, r.Name, c.IP(), msg)
		return nil, &accessDeniedError{msg: msg}
	}

	// Dibaca middleware Activity History supaya login tercatat atas nama user ini.
	c.Locals(constant.CtxUserID, u.ID)
	c.Locals(constant.CtxRoleName, r.Name)

	refresh, expiry, err := h.jwtSvc.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, err
	}

	device, ip, location := h.buildSessionInfo(c)
	now := time.Now()
	session := &model.RefreshToken{
		UserID:         u.ID,
		TokenHash:      hashToken(refresh),
		UserAgent:      c.Get("User-Agent"),
		Browser:        device.Browser,
		BrowserVersion: device.BrowserVersion,
		OS:             device.OS,
		OSVersion:      device.OSVersion,
		DeviceType:     string(device.DeviceType),
		IPAddress:      ip,
		Location:       location,
		ExpiresAt:      expiry,
		LastActiveAt:   &now,
	}
	if err := h.authRepo.SaveRefreshToken(session); err != nil {
		return nil, err
	}

	access, err := h.jwtSvc.GenerateAccessToken(u.ID, u.RoleID, session.ID, r.Name)
	if err != nil {
		return nil, err
	}
	if err := h.userRepo.UpdateLastLogin(u.ID); err != nil {
		log.Printf("auth: gagal update last_login_at untuk user %d: %v", u.ID, err)
	}

	return &LoginResponse{
		TokenType:    "Bearer",
		AccessToken:  access,
		RefreshToken: refresh,
		User: &UserSummary{
			ID: u.ID, Username: u.Username, Email: u.Email, RoleID: u.RoleID, RoleName: r.Name,
		},
		Session: &SessionInfo{
			ID:             session.ID,
			Browser:        device.Browser,
			BrowserVersion: device.BrowserVersion,
			OS:             device.OS,
			OSVersion:      device.OSVersion,
			DeviceType:     string(device.DeviceType),
			IPAddress:      ip,
			Location:       location,
			IsCurrent:      true,
		},
	}, nil
}

// resolveRegisterRoleName menentukan role akun baru saat self-register.
// Pendaftar bebas memilih super_admin, admin, atau karyawan. Kosong dianggap
// karyawan; nama role yang tidak dikenal ditolak. Pembatasan lokasi untuk
// admin/karyawan tetap berlaku lewat middleware.AccessAllowsRole.
func (h *Controller) resolveRegisterRoleName(requestedRole string) (string, error) {
	requestedRole = strings.ToLower(strings.TrimSpace(requestedRole))
	switch requestedRole {
	case "":
		return constant.RoleKaryawan, nil
	case constant.RoleKaryawan, constant.RoleAdmin, constant.RoleSuperAdmin:
		return requestedRole, nil
	default:
		return "", fmt.Errorf("role '%s' tidak dikenal", requestedRole)
	}
}

func (h *Controller) CheckUsernameAvailability(c *fiber.Ctx) error {
	username := strings.TrimSpace(c.Query("username"))
	if len(username) < 4 {
		return utils.OK(c, "username terlalu pendek", fiber.Map{"available": false})
	}
	_, err := h.userRepo.FindByUsername(username)
	available := err != nil
	return utils.OK(c, "berhasil cek ketersediaan username", fiber.Map{"available": available})
}

func (h *Controller) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	if req.CaptchaToken != "" || req.CaptchaAnswer != "" {
		if err := h.captchaSvc.Verify(req.CaptchaToken, req.CaptchaAnswer); err != nil {
			return utils.Fail(c, fiber.StatusBadRequest, "verifikasi captcha gagal: "+err.Error(), nil)
		}
	}

	if _, err := h.userRepo.FindByUsername(req.Username); err == nil {
		return utils.Fail(c, fiber.StatusConflict, "username sudah digunakan", nil)
	}
	if req.Email != "" {
		if _, err := h.userRepo.FindByEmail(req.Email); err == nil {
			return utils.Fail(c, fiber.StatusConflict, "email sudah terdaftar", nil)
		}
	}

	roleName, roleErr := h.resolveRegisterRoleName(req.RoleName)
	if roleErr != nil {
		return utils.Fail(c, fiber.StatusForbidden, roleErr.Error(), nil)
	}
	targetRole, err := h.roleRepo.FindByName(roleName)
	if err != nil {
		log.Printf("auth: role '%s' belum ada di database, jalankan seeder role dulu: %v", roleName, err)
		return utils.Fail(c, fiber.StatusInternalServerError, "pendaftaran belum bisa diproses, hubungi administrator", nil)
	}

	// Jangan buat akun yang tidak akan pernah bisa dipakai dari lokasi ini.
	if ok, msg := middleware.AccessAllowsRole(c, targetRole.Name); !ok {
		return utils.Fail(c, fiber.StatusForbidden, msg, nil)
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengenkripsi password", nil)
	}

	newUser := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashed,
		FullName:     req.FullName,
		PhoneNumber:  req.PhoneNumber,
		RoleID:       targetRole.ID,
		IsActive:     true,
		Is2FAEnabled: false,
	}
	if err := h.userRepo.Create(newUser); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat akun", nil)
	}

	res, err := h.issueTokens(c, newUser)
	if err != nil {
		return failIssueTokens(c, err, "akun berhasil dibuat, tapi gagal memulai sesi. Silakan login manual")
	}
	return utils.Created(c, "akun berhasil dibuat", res)
}

func (h *Controller) ForgotPassword(c *fiber.Ctx) error {
	var req ForgotPasswordRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	if err := h.humanCheckSvc.Verify(req.HumanCheckToken); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "verifikasi gagal: "+err.Error(), nil)
	}

	const resetDenied = "reset password tidak dapat diproses, username/email tidak ditemukan atau akun nonaktif"
	u, err := h.userRepo.FindByUsername(req.Identifier)
	if err != nil {
		u, err = h.userRepo.FindByEmail(req.Identifier)
	}
	if err != nil || !u.IsActive {
		return utils.Fail(c, fiber.StatusForbidden, resetDenied, nil)
	}
	hashed, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengenkripsi password baru", nil)
	}
	u.PasswordHash = hashed
	if err := h.userRepo.Update(u); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan password baru", nil)
	}

	if err := h.authRepo.RevokeAllUserTokens(u.ID); err != nil {
		log.Printf("auth: gagal mencabut sesi lama setelah reset password (alur biasa) untuk user %d: %v", u.ID, err)
	}

	return utils.OK(c, "password berhasil diubah, silakan login dengan password baru", nil)
}

func (h *Controller) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	u, err := h.userRepo.FindByUsername(req.Username)
	if h.loginAuthFailure(c, u, err, req.Password) {
		return nil
	}
	if h.loginLockedOrInactive(c, u) {
		return nil
	}
	if h.loginOutsideGudang(c, u) {
		return nil
	}
	if err := h.userRepo.ResetFailedLogin(u.ID); err != nil {
		log.Printf("auth: gagal reset failed_login_attempts untuk user %d: %v", u.ID, err)
	}

	if !u.Is2FAEnabled {
		res, err := h.issueTokens(c, u)
		if err != nil {
			return failIssueTokens(c, err, failedLoginProccessMessage)
		}
		return utils.OK(c, "login berhasil", res)
	}

	pendingToken, _, err := h.jwtSvc.GenerateRefreshToken(u.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memproses login", nil)
	}
	return utils.OK(c, "login berhasil, silakan verifikasi kode OTP", LoginResponse{
		RequireOTP: true, PendingToken: pendingToken,
	})
}

func (h *Controller) loginAuthFailure(c *fiber.Ctx, u *model.User, err error, password string) bool {
	if err != nil || !utils.ComparePassword(u.PasswordHash, password) {
		if u != nil {
			if err := h.userRepo.RegisterFailedLogin(u.ID, maxFailedLoginAttempts, lockDuration); err != nil {
				log.Printf("auth: gagal mencatat percobaan login gagal untuk user %d: %v", u.ID, err)
			}
		}
		_ = utils.Fail(c, fiber.StatusUnauthorized, "username atau password salah", nil)
		return true
	}
	return false
}

func (h *Controller) loginLockedOrInactive(c *fiber.Ctx, u *model.User) bool {
	if u.LockedUntil != nil && u.LockedUntil.After(time.Now()) {
		_ = utils.Fail(c, fiber.StatusTooManyRequests,
			"akun anda dikunci sementara karena terlalu banyak percobaan gagal, coba lagi setelah 15 menit", nil)
		return true
	}
	if !u.IsActive {
		_ = utils.Fail(c, fiber.StatusForbidden, accountInactiveMessage, nil)
		return true
	}
	return false
}

// loginOutsideGudang menolak login akun admin/karyawan yang datang dari luar
// area gudang (titik koordinat di Manajemen Gudang, plus daftar jaringan gudang
// kalau diatur). Dicek setelah password terverifikasi supaya pesannya tidak
// bocor ke orang yang tidak punya kredensial. Super admin tidak dibatasi.
func (h *Controller) loginOutsideGudang(c *fiber.Ctx, u *model.User) bool {
	r, err := h.roleRepo.FindByID(u.RoleID)
	if err != nil {
		// Fail-closed: role tidak terbaca -> jangan lanjutkan login.
		_ = utils.Fail(c, fiber.StatusInternalServerError, "gagal memproses login", nil)
		return true
	}
	ok, msg := middleware.AccessAllowsRole(c, r.Name)
	if ok {
		return false
	}
	log.Printf("auth: login '%s' (%s) ditolak dari IP %s: %s", u.Username, r.Name, c.IP(), msg)
	_ = utils.Fail(c, fiber.StatusForbidden, msg, nil)
	return true
}

func (h *Controller) StartTwoFactorSetup(c *fiber.Ctx) error {
	userID, ok := c.Locals(constant.CtxUserID).(uint)
	if !ok {
		return utils.Fail(c, fiber.StatusUnauthorized, "sesi tidak valid", nil)
	}
	u, err := h.userRepo.FindByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrUserNotFound, nil)
	}
	if u.Is2FAEnabled {
		return utils.Fail(c, fiber.StatusBadRequest, "2FA sudah aktif untuk akun ini", nil)
	}

	pendingToken, _, err := h.jwtSvc.GenerateRefreshToken(u.ID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memulai setup 2FA", nil)
	}
	return utils.OK(c, "silakan lanjutkan setup 2FA", fiber.Map{"pending_token": pendingToken})
}

func (h *Controller) Setup2FA(c *fiber.Ctx) error {
	var req Setup2FARequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	claims, err := h.jwtSvc.ParseRefreshToken(req.PendingToken)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, constant.ErrSessionExpired, nil)
	}

	u, err := h.userRepo.FindByID(utils.ParseUintSubject(claims.Subject))
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrUserNotFound, nil)
	}
	// pendingToken di sini adalah JWT stateless yang sama bentuknya dengan
	// pendingToken yang diterbitkan Login() saat akun sudah punya 2FA aktif
	// (dua-duanya lewat GenerateRefreshToken, tidak pernah disimpan ke tabel
	// refresh_tokens, jadi tidak bisa dibedakan "asalnya" cuma dari token
	// itu sendiri). Tanpa guard ini, siapa pun yang cuma tahu password akun
	// (misalnya dari kebocoran password di layanan lain) bisa Login untuk
	// mendapat pendingToken, lalu pakai endpoint setup 2FA ini buat
	// memasang secret TOTP PILIHAN SENDIRI dan langsung lolos verifikasi —
	// sama sekali tidak butuh HP/authenticator korban. Itu meniadakan
	// gunanya 2FA. StartTwoFactorSetup (jalur resmi, perlu login dulu) sudah
	// menolak kalau 2FA aktif; guard yang sama wajib ada di sini juga karena
	// endpoint ini tidak mensyaratkan JWTAuth sama sekali.
	if u.Is2FAEnabled {
		return utils.Fail(c, fiber.StatusBadRequest, "2FA sudah aktif untuk akun ini, matikan dulu lewat administrator sebelum setup ulang", nil)
	}

	totpLabel := u.Email
	if totpLabel == "" {
		totpLabel = u.Username
	}
	secret, qr, err := utils.GenerateTOTPSecret(h.totpIssuer, totpLabel)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal membuat QR 2FA", nil)
	}
	return utils.OK(c, "silakan scan QR dengan Google Authenticator", Setup2FAResponse{
		Secret: secret, QRCodePNG: qr,
	})
}

func (h *Controller) ConfirmSetup2FA(c *fiber.Ctx) error {
	var req ConfirmSetup2FARequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	claims, err := h.jwtSvc.ParseRefreshToken(req.PendingToken)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, constant.ErrSessionExpired, nil)
	}
	userID := utils.ParseUintSubject(claims.Subject)

	u, err := h.userRepo.FindByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrUserNotFound, nil)
	}
	// Guard yang sama seperti di Setup2FA (lihat komentar di sana): tanpa
	// ini, pendingToken dari Login (yang cuma butuh password) bisa dipakai
	// buat menimpa TOTPSecret akun yang 2FA-nya SUDAH aktif dengan secret
	// pilihan penyerang sendiri, lalu langsung dapat access/refresh token
	// via issueTokens di bawah — bypass total 2FA tanpa pernah menyentuh
	// authenticator korban.
	if u.Is2FAEnabled {
		return utils.Fail(c, fiber.StatusBadRequest, "2FA sudah aktif untuk akun ini, matikan dulu lewat administrator sebelum setup ulang", nil)
	}

	if !utils.VerifyTOTP(req.OTPCode, req.Secret) {
		return utils.Fail(c, fiber.StatusBadRequest, "Kode OTP salah atau sudah kedaluwarsa", nil)
	}
	if !h.otpReplayGuard.checkAndMark(userID, req.OTPCode) {
		return utils.Fail(c, fiber.StatusBadRequest, otpCodeReusedMessage, nil)
	}
	if err := h.userRepo.UpdateTOTPSecret(userID, req.Secret, true); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mengaktifkan 2FA", nil)
	}
	u.Is2FAEnabled = true
	res, err := h.issueTokens(c, u)
	if err != nil {
		return failIssueTokens(c, err, "gagal menerbitkan sesi")
	}
	return utils.OK(c, "Two Factor Authentication berhasil diaktifkan", res)
}

func (h *Controller) verifyOTPCode(req VerifyOTPRequest, totpSecret string) bool {
	return utils.VerifyTOTP(req.OTPCode, totpSecret)
}

func (h *Controller) VerifyOTP(c *fiber.Ctx) error {
	var req VerifyOTPRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	claims, err := h.jwtSvc.ParseRefreshToken(req.PendingToken)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, constant.ErrSessionExpired, nil)
	}

	userID := utils.ParseUintSubject(claims.Subject)
	u, err := h.userRepo.FindByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrUserNotFound, nil)
	}
	// Tanpa ini kode OTP 6 digit bisa ditebak tanpa batas selama pendingToken
	// masih berlaku. Pakai penghitung gagal-login yang sama dengan password.
	if h.loginLockedOrInactive(c, u) {
		return nil
	}
	if h.loginOutsideGudang(c, u) {
		return nil
	}
	if !h.verifyOTPCode(req, u.TOTPSecret) {
		if rerr := h.userRepo.RegisterFailedLogin(u.ID, maxFailedLoginAttempts, lockDuration); rerr != nil {
			log.Printf("auth: gagal mencatat OTP salah untuk user %d: %v", u.ID, rerr)
		}
		return utils.Fail(c, fiber.StatusBadRequest, "Kode OTP salah atau sudah kedaluwarsa", nil)
	}
	if !h.otpReplayGuard.checkAndMark(u.ID, req.OTPCode) {
		return utils.Fail(c, fiber.StatusBadRequest, "Kode OTP ini sudah pernah dipakai, silakan gunakan kode baru", nil)
	}

	if rerr := h.userRepo.ResetFailedLogin(u.ID); rerr != nil {
		log.Printf("auth: gagal reset penghitung gagal-login user %d: %v", u.ID, rerr)
	}
	res, err := h.issueTokens(c, u)
	if err != nil {
		return failIssueTokens(c, err, "gagal menerbitkan sesi")
	}
	return utils.OK(c, "Verifikasi OTP Berhasil", res)
}

func (h *Controller) RefreshToken(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if !utils.ParseAndValidate(c, &req) {
		return nil
	}

	claims, err := h.jwtSvc.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "refresh token tidak valid atau kedaluwarsa", nil)
	}

	userID := utils.ParseUintSubject(claims.Subject)
	oldSession, err := h.authRepo.FindActiveRefreshToken(userID, hashToken(req.RefreshToken))
	if err != nil {
		return utils.Fail(c, fiber.StatusUnauthorized, "sesi tidak ditemukan atau sudah di-revoke", nil)
	}

	u, err := h.userRepo.FindByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrUserNotFound, nil)
	}
	// Akun yang dinonaktifkan tidak boleh memperpanjang sesi lewat refresh token.
	if !u.IsActive {
		return utils.Fail(c, fiber.StatusForbidden, accountInactiveMessage, nil)
	}

	res, err := h.issueTokens(c, u)
	if err != nil {
		return failIssueTokens(c, err, "gagal memperbarui token")
	}
	// issueTokens membuat BARIS SESI BARU (rotasi token, dengan LastActiveAt
	// = sekarang). Cabut baris lama supaya tidak menumpuk baris "aktif"
	// duplikat untuk device yang sama di Riwayat Login, dan supaya kalau
	// admin sempat mencabut baris lama ini SEBELUM sempat di-refresh,
	// refresh berikutnya otomatis ditolak (lihat pengecekan "revoked =
	// false" di FindActiveRefreshToken).
	if err := h.authRepo.RevokeSession(userID, oldSession.ID, userID); err != nil {
		log.Printf("auth: gagal mencabut sesi lama %d setelah refresh: %v", oldSession.ID, err)
	}
	return utils.OK(c, "token berhasil diperbarui", res)
}

func (h *Controller) Logout(c *fiber.Ctx) error {
	userID, _ := c.Locals(constant.CtxUserID).(uint)
	if err := h.authRepo.RevokeAllUserTokens(userID); err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal logout", nil)
	}
	return utils.OK(c, "logout berhasil", nil)
}

func toSessionInfo(s model.RefreshToken, currentSessionID uint) SessionInfo {
	lastActiveAt := ""
	if s.LastActiveAt != nil {
		lastActiveAt = s.LastActiveAt.Format(time.RFC3339)
	}
	return SessionInfo{
		ID:             s.ID,
		Browser:        s.Browser,
		BrowserVersion: s.BrowserVersion,
		OS:             s.OS,
		OSVersion:      s.OSVersion,
		DeviceType:     s.DeviceType,
		IPAddress:      s.IPAddress,
		Location:       s.Location,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
		LastActiveAt:   lastActiveAt,
		IsCurrent:      currentSessionID != 0 && s.ID == currentSessionID,
	}
}

func (h *Controller) ListSessions(c *fiber.Ctx) error {
	userID, ok := c.Locals(constant.CtxUserID).(uint)
	if !ok {
		return utils.Fail(c, fiber.StatusUnauthorized, constant.ErrInvalidToken, nil)
	}
	currentSessionID, _ := c.Locals(constant.CtxSessionID).(uint)

	sessions, err := h.authRepo.ListActiveSessions(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memuat daftar sesi", nil)
	}

	result := make([]SessionInfo, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, toSessionInfo(s, currentSessionID))
	}
	return utils.OK(c, "berhasil memuat daftar sesi aktif", SessionListResponse{Sessions: result})
}

func (h *Controller) RevokeSession(c *fiber.Ctx) error {
	userID, ok := c.Locals(constant.CtxUserID).(uint)
	if !ok {
		return utils.Fail(c, fiber.StatusUnauthorized, constant.ErrInvalidToken, nil)
	}

	var params model.IDParam
	if err := c.ParamsParser(&params); err != nil {
		return utils.Fail(c, fiber.StatusBadRequest, "id sesi tidak valid", nil)
	}

	if err := h.authRepo.RevokeSession(userID, params.ID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return utils.Fail(c, fiber.StatusNotFound, "sesi tidak ditemukan", nil)
		}
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal mencabut sesi", nil)
	}

	currentSessionID, _ := c.Locals(constant.CtxSessionID).(uint)
	revokedCurrent := currentSessionID != 0 && currentSessionID == params.ID
	return utils.OK(c, "sesi berhasil dicabut", fiber.Map{"revoked_current": revokedCurrent})
}

func (h *Controller) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals(constant.CtxUserID).(uint)
	if !ok {
		return utils.Fail(c, fiber.StatusUnauthorized, constant.ErrInvalidToken, nil)
	}

	u, err := h.userRepo.FindByID(userID)
	if err != nil {
		return utils.Fail(c, fiber.StatusNotFound, constant.ErrUserNotFound, nil)
	}
	r, err := h.roleRepo.FindByID(u.RoleID)
	if err != nil {
		return utils.Fail(c, fiber.StatusInternalServerError, "gagal memuat data role", nil)
	}

	return utils.OK(c, "berhasil membaca sesi aktif", MeResponse{
		ID:           u.ID,
		Username:     u.Username,
		Email:        u.Email,
		FullName:     u.FullName,
		PhoneNumber:  u.PhoneNumber,
		RoleID:       u.RoleID,
		RoleName:     r.Name,
		Is2FAEnabled: u.Is2FAEnabled,
		AvatarURL:    u.AvatarURL(),
	})
}

func (h *Controller) RegisterRoutes(router fiber.Router) {
	g := router.Group("/auth")

	g.Get("/username-available", middleware.UsernameCheckRateLimiter(), h.CheckUsernameAvailability)
	g.Post("/register", middleware.RegisterRateLimiter(), h.Register)
	g.Post("/login", h.Login)
	g.Post("/2fa/setup", h.Setup2FA)
	g.Post("/2fa/confirm", h.ConfirmSetup2FA)
	g.Post("/verify-otp", h.VerifyOTP)
	g.Post("/refresh", h.RefreshToken)

	g.Post("/password/forgot", middleware.PasswordResetRateLimiter(), h.ForgotPassword)

	protected := g.Group("/", middleware.JWTAuth(h.jwtSvc))
	protected.Post("/logout", h.Logout)
	protected.Get("/me", h.Me)
	protected.Post("/2fa/start", h.StartTwoFactorSetup)
	protected.Get("/sessions", h.ListSessions)
	protected.Delete("/sessions/:id", h.RevokeSession)
}
