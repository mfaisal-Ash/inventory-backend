package utils

import "errors"

var (
	ErrNotFound          = errors.New("data tidak ditemukan")
	ErrAlreadyExists     = errors.New("data sudah ada")
	ErrInvalidCredential = errors.New("username atau password salah")
	ErrInvalidOTP        = errors.New("kode OTP salah atau sudah kedaluwarsa")
	ErrUnauthorized      = errors.New("kamu tidak memiliki akses")
	ErrForbiddenModule   = errors.New("role kamu tidak memiliki izin mengakses modul ini")
	ErrTokenExpired      = errors.New("token sudah kadaluarsa, coba login kembali")
	ErrAccountInactive   = errors.New("akun kamu tidak aktif, hubungi superadmin")
)
