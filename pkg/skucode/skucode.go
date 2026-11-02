// Package skucode membentuk kode barang (SKU) dengan format:
//
//	<KATEGORI>-<PRODUK>-<DDMMYYYY>-<NOMOR URUT>
//
// Contoh (31 Oktober 2026): OP-ONT-31102026-01, OP-MODEM-31102026-02,
// ELK-KABEL-31102026-01.
//
//   - KATEGORI : singkatan kategori (kolom kategori.singkatan; kalau kosong
//     diturunkan dari nama kategori, lihat KategoriCode).
//   - PRODUK   : singkatan nama barang (lihat ProdukCode).
//   - DDMMYYYY : TANGGAL CATAT yang dipilih pengguna di form (bukan jam
//     server), jadi data susulan bertanggal mundur (mis. 5 Oktober 2026)
//     mendapat kode bertanggal 05102026.
//   - NOMOR    : urut per KATEGORI di tanggal itu (minimal 2 digit) — ganti
//     kategori atau ganti tanggal, mulai lagi dari 01. Produk berbeda dalam
//     satu kategori di tanggal yang sama melanjutkan nomor yang sama
//     (OP-MODEM-31102026-01 lalu OP-ROUTER-31102026-02).
//
// Kode lama berformat <KATEGORI>-<PRODUK>-<MMYYYY>-<NOMOR> tetap valid dan
// tidak diubah; hanya kode baru yang memakai format tanggal lengkap.
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
	// kode_barang (size:30): 6+1+6+1+8+1+4 = 27.
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

// Periode memformat TANGGAL LENGKAP sebagai DDMMYYYY (31 Oktober 2026 ->
// "31102026"). Nama fungsi dipertahankan supaya pemanggil lama tetap
// kompatibel; isinya sekarang tanggal, bukan lagi bulan+tahun.
func Periode(t time.Time) string { return t.Format("02012006") }

// Prefix mengembalikan "<KATEGORI>-<PRODUK>-<DDMMYYYY>" (tanpa nomor urut).
func Prefix(kategoriCode, produkCode, periode string) string {
	return kategoriCode + "-" + produkCode + "-" + periode
}

// Format menempelkan nomor urut (minimal 2 digit) ke prefix.
func Format(prefix string, nomor int) string {
	return fmt.Sprintf("%s-%02d", prefix, nomor)
}

// LikePattern dipakai untuk menyaring kode_barang satu kategori+tanggal di DB.
func LikePattern(kategoriCode, periode string) string {
	return kategoriCode + "-%-" + periode + "-%"
}

// MaxNomor mencari nomor urut terbesar dari daftar kode yang berformat
// <kategoriCode>-<apa pun>-<DDMMYYYY>-<angka>. Kode berformat lain (termasuk
// kode lama ber-MMYYYY) diabaikan.
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

// ParseTanggal membaca tanggal catat dari form (format YYYY-MM-DD). Kosong =
// hari ini. Tanggal di masa depan ditolak: stok baru tidak boleh dicatat
// "dari masa depan" karena merusak perhitungan neraca.
func ParseTanggal(raw string, now time.Time) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if raw == "" {
		return today, nil
	}
	if len(raw) > 10 {
		raw = raw[:10] // terima juga ISO penuh "2026-10-05T00:00:00Z"
	}
	t, err := time.ParseInLocation("2006-01-02", raw, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("format tanggal harus YYYY-MM-DD")
	}
	if t.After(today) {
		return time.Time{}, fmt.Errorf("tanggal tidak boleh lebih dari hari ini")
	}
	return t, nil
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
