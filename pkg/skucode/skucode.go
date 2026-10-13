// Package skucode membentuk kode barang (SKU) dengan format:
//
//	<KATEGORI>-<PRODUK>-<MMYYYY>-<NOMOR URUT>
//
// Contoh (Oktober 2026): OP-MODEM-102026-001, OP-MODEM-102026-002,
// ELK-KABEL-102026-001.
//
//   - KATEGORI : singkatan kategori (kolom kategori.singkatan; kalau kosong
//     diturunkan dari nama kategori, lihat KategoriCode).
//   - PRODUK   : singkatan nama barang (lihat ProdukCode).
//   - MMYYYY   : periode bulan+tahun saat kode dibuat (real-time).
//   - NOMOR    : urut per KATEGORI di periode itu — ganti kategori, mulai
//     lagi dari 001. Produk berbeda dalam satu kategori melanjutkan nomor
//     yang sama (OP-MODEM-102026-002 lalu OP-ROUTER-102026-003).
package skucode

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	// Batas panjang tiap segmen supaya total kode muat di kolom
	// kode_barang (size:30): 6+1+6+1+6+1+4 = 25.
	MaxKategoriLen = 6
	MaxProdukLen   = 6

	fallbackKategori = "BRG"
	fallbackProduk   = "BRG"
)

// TipeONT adalah pilihan tipe (warna port) khusus barang ONT.
var TipeONT = []string{"Hijau", "Biru"}

// alnumUpper membuang semua karakter selain huruf/angka dan mengubahnya ke
// huruf besar, supaya aman dipakai sebagai segmen kode (tanpa "-" atau spasi).
func alnumUpper(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// KategoriCode memakai singkatan kategori bila ada; kalau kosong, diturunkan
// dari nama kategori: nama satu kata -> 3 huruf pertama, nama beberapa kata
// -> inisial tiap kata (maks 3).
func KategoriCode(singkatan, nama string) string {
	if c := truncate(alnumUpper(singkatan), MaxKategoriLen); c != "" {
		return c
	}
	words := strings.Fields(nama)
	var code string
	if len(words) > 1 {
		var b strings.Builder
		for _, w := range words {
			if a := alnumUpper(w); a != "" {
				b.WriteRune([]rune(a)[0])
			}
			if len([]rune(b.String())) == 3 {
				break
			}
		}
		code = b.String()
	} else {
		code = truncate(alnumUpper(nama), 3)
	}
	if code == "" {
		return fallbackKategori
	}
	return code
}

// ProdukCode = singkatan nama barang: kata pertama nama barang (huruf/angka
// saja), maks MaxProdukLen karakter. "Modem ZTE F670L" -> "MODEM".
func ProdukCode(nama string) string {
	for _, w := range strings.Fields(nama) {
		if a := alnumUpper(w); a != "" {
			return truncate(a, MaxProdukLen)
		}
	}
	return fallbackProduk
}

// Periode memformat bulan+tahun sebagai MMYYYY (Oktober 2026 -> "102026").
func Periode(t time.Time) string { return t.Format("012006") }

// Prefix mengembalikan "<KATEGORI>-<PRODUK>-<MMYYYY>" (tanpa nomor urut).
func Prefix(kategoriCode, produkCode, periode string) string {
	return kategoriCode + "-" + produkCode + "-" + periode
}

// Format menempelkan nomor urut (minimal 3 digit) ke prefix.
func Format(prefix string, nomor int) string {
	return fmt.Sprintf("%s-%03d", prefix, nomor)
}

// LikePattern dipakai untuk menyaring kode_barang satu kategori+periode di DB.
func LikePattern(kategoriCode, periode string) string {
	return kategoriCode + "-%-" + periode + "-%"
}

// MaxNomor mencari nomor urut terbesar dari daftar kode yang berformat
// <kategoriCode>-<apa pun>-<periode>-<angka>. Kode berformat lain diabaikan.
func MaxNomor(codes []string, kategoriCode, periode string) int {
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(kategoriCode) + `-[A-Z0-9]+-` + regexp.QuoteMeta(periode) + `-(\d+)$`)
	max := 0
	for _, c := range codes {
		m := re.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(c)))
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[1]); err == nil && n > max {
			max = n
		}
	}
	return max
}

var ontWord = regexp.MustCompile(`(?i)(^|[^a-z0-9])ont($|[^a-z0-9])`)

// IsONT true bila nama barang memuat kata "ONT" (mis. "ONT Huawei HG8245").
func IsONT(nama string) bool { return ontWord.MatchString(nama) }

// NormalizeTipeONT menerima "hijau", "Port Hijau", "BIRU" dst. dan
// mengembalikan "Hijau"/"Biru"; ok=false kalau bukan salah satunya.
func NormalizeTipeONT(tipe string) (string, bool) {
	t := strings.ToLower(strings.TrimSpace(tipe))
	t = strings.TrimSpace(strings.TrimPrefix(t, "port"))
	for _, v := range TipeONT {
		if strings.ToLower(v) == t {
			return v, true
		}
	}
	return "", false
}
