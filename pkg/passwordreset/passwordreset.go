// Package passwordreset mengimplementasikan alur "lupa password" yang aman:
// server men-generate kode OTP acak, mengirimkannya lewat WhatsApp ke nomor
// HP yang SUDAH terdaftar untuk akun tersebut, lalu pengguna harus mengetik
// ulang kode itu sebelum boleh mengganti password.
//
// Sebelum paket ini dibuat, endpoint reset password cuma mensyaratkan
// identifier (username/email) + lolos human-check (captcha) — TANPA
// pembuktian bahwa yang meminta benar-benar pemilik akun. Siapa pun yang
// tahu/menebak username korban bisa langsung mengganti passwordnya dan
// mengambil alih akun (termasuk akun super_admin). Paket ini menutup celah
// itu dengan mengikat proses reset ke bukti kepemilikan nomor HP terdaftar.
//
// Desainnya stateless & mengikuti pola pkg/humancheck yang sudah ada di
// proyek ini (HMAC-signed ticket + in-memory replay guard), supaya tidak
// perlu migrasi skema database baru:
//  1. IssueTicket(userID, code) dipanggil SETELAH kode berhasil dikirim via
//     WhatsApp — mengembalikan "tiket" berisi userID + hash kode (bukan
//     kode aslinya) yang ditandatangani HMAC dan diberikan ke client.
//  2. Client memasukkan kode yang diterima via WhatsApp, dikirim bersama
//     tiket ke VerifyTicket — server mencocokkan hash, memeriksa masa
//     berlaku & bahwa tiket belum pernah dipakai, lalu mengembalikan userID
//     yang boleh diproses reset password-nya.
package passwordreset

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

var (
	ErrInvalid      = errors.New("tiket reset tidak valid atau rusak")
	ErrExpired      = errors.New("kode verifikasi sudah kedaluwarsa, silakan minta kode baru")
	ErrAlreadyUsed  = errors.New("tiket reset ini sudah dipakai, silakan minta kode baru")
	ErrCodeMismatch = errors.New("kode verifikasi salah")
)

const (
	codeLength   = 6
	payloadBytes = 8 + 4 + 8 + 8 // issuedAt + userID + codeHash(truncated) + nonce
)

// GenerateCode menghasilkan kode OTP numerik acak sepanjang codeLength
// digit, dipakai sebagai isi pesan WhatsApp yang dikirim ke pengguna.
func GenerateCode() (string, error) {
	max := big.NewInt(1)
	for i := 0; i < codeLength; i++ {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", codeLength, n), nil
}

type usedStore struct {
	mu   sync.Mutex
	data map[string]time.Time
}

func newUsedStore() *usedStore {
	s := &usedStore{data: make(map[string]time.Time)}
	go s.cleanupLoop()
	return s
}

func (s *usedStore) isUsed(ticket string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[ticket]
	return ok
}

func (s *usedStore) markUsed(ticket string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[ticket] = time.Now().Add(ttl)
}

func (s *usedStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for ticket, exp := range s.data {
			if now.After(exp) {
				delete(s.data, ticket)
			}
		}
		s.mu.Unlock()
	}
}

type Service struct {
	secret []byte
	ttl    time.Duration
	used   *usedStore
}

// NewService: ttlMinutes <= 0 jatuh balik ke 10 menit supaya konfigurasi
// yang kosong/salah tidak membuat tiket berlaku selamanya (fail-safe, sama
// semangatnya dengan validateProductionSecrets di pkg/config).
func NewService(secret string, ttlMinutes int) *Service {
	if ttlMinutes <= 0 {
		ttlMinutes = 10
	}
	return &Service{
		secret: []byte(secret),
		ttl:    time.Duration(ttlMinutes) * time.Minute,
		used:   newUsedStore(),
	}
}

// codeHash HARUS di-keying pakai secret milik Service (bukan sesuatu yang
// publik seperti userID) — tiket yang beredar ke client memuat userID
// dalam bentuk terang plus hash 8-byte ini. Kode OTP cuma 6 digit (1 juta
// kemungkinan), jadi kalau hash-nya bisa dihitung ulang tanpa secret,
// siapa pun yang memegang tiket bisa brute-force ke-1 juta kombinasi
// dalam hitungan milidetik dan menemukan kodenya TANPA pernah menerima
// pesan WhatsApp-nya — meniadakan gunanya verifikasi ini. Dengan secret
// ikut jadi key HMAC, brute force itu mustahil tanpa membobol secret
// server (yang kalau sudah bocor, tiket bisa dipalsukan langsung juga).
func (s *Service) codeHash(userID uint, code string) []byte {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(fmt.Sprintf("code:%d:%s", userID, code)))
	return mac.Sum(nil)[:8]
}

// IssueTicket membuat tiket ber-signature untuk userID+code yang SUDAH
// dikirim ke pengguna lewat WhatsApp. Tiket ini yang dipegang client
// (bukan kode aslinya) sampai langkah verifikasi.
func (s *Service) IssueTicket(userID uint, code string) (string, error) {
	payload := make([]byte, 0, payloadBytes)

	issuedAt := make([]byte, 8)
	binary.BigEndian.PutUint64(issuedAt, uint64(time.Now().Unix()))
	payload = append(payload, issuedAt...)

	uid := make([]byte, 4)
	binary.BigEndian.PutUint32(uid, uint32(userID))
	payload = append(payload, uid...)

	payload = append(payload, s.codeHash(userID, code)...)

	nonce := make([]byte, 8)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	payload = append(payload, nonce...)

	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	sig := mac.Sum(nil)

	return base64.RawURLEncoding.EncodeToString(append(payload, sig...)), nil
}

// VerifyTicket memeriksa signature, masa berlaku, replay, dan kecocokan
// kode yang diketik pengguna terhadap hash yang tertanam di tiket. Kalau
// semuanya valid, mengembalikan userID yang boleh diproses reset
// password-nya — TAPI belum menandai tiket sebagai "dipakai"; panggil
// Consume setelah password benar-benar berhasil diganti, supaya tiket yang
// gagal disimpan (mis. error DB) masih bisa dicoba ulang oleh pengguna
// tanpa perlu minta kode baru.
func (s *Service) VerifyTicket(ticket, code string) (uint, error) {
	raw, err := base64.RawURLEncoding.DecodeString(ticket)
	if err != nil || len(raw) != payloadBytes+sha256.Size {
		return 0, ErrInvalid
	}
	payload := raw[:payloadBytes]
	sig := raw[payloadBytes:]

	mac := hmac.New(sha256.New, s.secret)
	mac.Write(payload)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return 0, ErrInvalid
	}

	if s.used.isUsed(ticket) {
		return 0, ErrAlreadyUsed
	}

	issuedAt := time.Unix(int64(binary.BigEndian.Uint64(payload[0:8])), 0)
	if time.Since(issuedAt) > s.ttl {
		return 0, ErrExpired
	}

	userID := uint(binary.BigEndian.Uint32(payload[8:12]))
	wantHash := payload[12:20]
	gotHash := s.codeHash(userID, code)
	if subtle.ConstantTimeCompare(wantHash, gotHash) != 1 {
		return 0, ErrCodeMismatch
	}

	return userID, nil
}

// Consume menandai tiket sebagai sudah dipakai supaya tidak bisa dipakai
// ulang untuk mengganti password berkali-kali (single-use).
func (s *Service) Consume(ticket string) {
	s.used.markUsed(ticket, s.ttl)
}
