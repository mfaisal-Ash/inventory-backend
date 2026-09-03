package auth

import (
	"time"

	authRepo "github.com/mfaisal-Ash/inventory-backend/internal/repositories/auth"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/role"
	"github.com/mfaisal-Ash/inventory-backend/internal/repositories/users"
	"github.com/mfaisal-Ash/inventory-backend/pkg/captcha"
	"github.com/mfaisal-Ash/inventory-backend/pkg/config"
	"github.com/mfaisal-Ash/inventory-backend/pkg/geoip"
	"github.com/mfaisal-Ash/inventory-backend/pkg/humancheck"
	"github.com/mfaisal-Ash/inventory-backend/pkg/passwordreset"
	"github.com/mfaisal-Ash/inventory-backend/pkg/utils"
	"github.com/mfaisal-Ash/inventory-backend/pkg/wa"
)

const otpReplayTTL = 5 * time.Minute

type Controller struct {
	userRepo      users.Repository
	roleRepo      role.Repository
	authRepo      authRepo.Repository
	jwtSvc        *utils.JWTService
	captchaSvc    *captcha.Service
	humanCheckSvc *humancheck.Service
	geoipSvc      geoip.Resolver

	waSender         wa.Sender
	passwordResetSvc *passwordreset.Service

	appEnv     string
	totpIssuer string

	otpReplayGuard *totpReplayGuard
}

type Params struct {
	UserRepo      users.Repository
	RoleRepo      role.Repository
	AuthRepo      authRepo.Repository
	JWTSvc        *utils.JWTService
	CaptchaSvc    *captcha.Service
	HumanCheckSvc *humancheck.Service
	GeoipSvc      geoip.Resolver
	Cfg           *config.Config

	WASender         wa.Sender
	PasswordResetSvc *passwordreset.Service
}

func New(p Params) *Controller {
	return &Controller{
		userRepo:      p.UserRepo,
		roleRepo:      p.RoleRepo,
		authRepo:      p.AuthRepo,
		jwtSvc:        p.JWTSvc,
		captchaSvc:    p.CaptchaSvc,
		humanCheckSvc: p.HumanCheckSvc,
		geoipSvc:      p.GeoipSvc,

		waSender:         p.WASender,
		passwordResetSvc: p.PasswordResetSvc,

		appEnv:     p.Cfg.App.Env,
		totpIssuer: p.Cfg.TOTP.Issuer,

		otpReplayGuard: newTOTPReplayGuard(otpReplayTTL),
	}
}

type RegisterRequest struct {
	Username             string `json:"username" validate:"required,min=4,max=50"`
	Email                string `json:"email" validate:"omitempty,email"`
	Password             string `json:"password" validate:"required,min=8"`
	PasswordConfirmation string `json:"password_confirmation" validate:"required,eqfield=Password"`
	FullName             string `json:"full_name" validate:"required"`
	PhoneNumber          string `json:"phone_number"`

	RoleName      string `json:"role_name"`
	CaptchaToken  string `json:"captcha_token" validate:"required"`
	CaptchaAnswer string `json:"captcha_answer" validate:"required"`
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

// RequestPasswordResetOTPRequest adalah langkah 1 dari alur lupa password:
// meminta server mengirim kode OTP via WhatsApp ke nomor HP yang terdaftar
// untuk akun dengan identifier ini.
type RequestPasswordResetOTPRequest struct {
	Identifier      string `json:"identifier" validate:"required"`
	HumanCheckToken string `json:"human_check_token" validate:"required"`
}

// ResetPasswordRequest adalah langkah 2: mengetik ulang kode OTP yang
// diterima via WhatsApp bersama reset ticket dari langkah 1, untuk
// membuktikan kepemilikan akun sebelum password diganti.
type ResetPasswordRequest struct {
	ResetTicket             string `json:"reset_ticket" validate:"required"`
	Code                    string `json:"code" validate:"required,len=6"`
	NewPassword             string `json:"new_password" validate:"required,min=8"`
	NewPasswordConfirmation string `json:"new_password_confirmation" validate:"required,eqfield=NewPassword"`
}

// ForgotPasswordRequest: alur lupa password "biasa" (satu langkah, TANPA
// kode OTP WhatsApp) — lihat catatan keamanan di Controller.ForgotPassword.
type ForgotPasswordRequest struct {
	Identifier              string `json:"identifier" validate:"required"`
	NewPassword             string `json:"new_password" validate:"required,min=8"`
	NewPasswordConfirmation string `json:"new_password_confirmation" validate:"required,eqfield=NewPassword"`
	HumanCheckToken         string `json:"human_check_token" validate:"required"`
}

type Setup2FARequest struct {
	PendingToken string `json:"pending_token" validate:"required"`
}

type ConfirmSetup2FARequest struct {
	PendingToken string `json:"pending_token" validate:"required"`
	Secret       string `json:"secret" validate:"required"`
	OTPCode      string `json:"otp_code" validate:"required,len=6"`
}

type VerifyOTPRequest struct {
	PendingToken string `json:"pending_token" validate:"required"`
	OTPCode      string `json:"otp_code" validate:"required,len=6"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type UserSummary struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	RoleID   uint   `json:"role_id"`
	RoleName string `json:"role_name"`
}

type SessionInfo struct {
	ID             uint   `json:"id"`
	Browser        string `json:"browser"`
	BrowserVersion string `json:"browser_version"`
	OS             string `json:"os"`
	OSVersion      string `json:"os_version"`
	DeviceType     string `json:"device_type"`
	IPAddress      string `json:"ip_address"`
	Location       string `json:"location"`
	CreatedAt      string `json:"created_at"`
	LastActiveAt   string `json:"last_active_at,omitempty"`
	IsCurrent      bool   `json:"is_current"`
}

type SessionListResponse struct {
	Sessions []SessionInfo `json:"sessions"`
}

type LoginResponse struct {
	RequireOTP   bool         `json:"require_otp"`
	PendingToken string       `json:"pending_token,omitempty"`
	TokenType    string       `json:"token_type,omitempty"`
	AccessToken  string       `json:"access_token,omitempty"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	User         *UserSummary `json:"user,omitempty"`
	Session      *SessionInfo `json:"session,omitempty"`
}

type RequestPasswordResetOTPResponse struct {
	ResetTicket   string `json:"reset_ticket"`
	MaskedPhone   string `json:"masked_phone"`
	ExpiresInMins int    `json:"expires_in_minutes"`
}

type Setup2FAResponse struct {
	Secret    string `json:"secret"`
	QRCodePNG string `json:"qr_code_png"`
}

type MeResponse struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	PhoneNumber  string `json:"phone_number"`
	RoleID       uint   `json:"role_id"`
	RoleName     string `json:"role_name"`
	Is2FAEnabled bool   `json:"is_2fa_enabled"`
	AvatarURL    string `json:"avatar_url"`
}
