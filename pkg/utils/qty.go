package utils

import (
	"strings"
	"unicode"
)

// FormatQty menggabungkan jumlah dan satuan untuk tampilan/cetak, mis.
// FormatQty("5", "Rim") -> "5 Rim", FormatQty("1", "Roll") -> "1 Roll".
//
// Data lama menyimpan qty pengajuan sebagai teks bebas yang sudah memuat
// satuan ("5 rim"); kalau qty sudah mengandung huruf, dikembalikan apa adanya
// supaya satuan tidak tercetak dua kali. Satuan kosong -> qty saja.
func FormatQty(qty, satuan string) string {
	qty = strings.TrimSpace(qty)
	satuan = strings.TrimSpace(satuan)
	if satuan == "" || qty == "" || qty == "-" {
		return qty
	}
	for _, r := range qty {
		if unicode.IsLetter(r) {
			return qty
		}
	}
	return qty + " " + satuan
}

// LeadingInt mengambil angka di awal teks jumlah ("5 Rim" -> 5, true).
// Dipakai untuk menjumlahkan kolom Qty yang sudah memuat satuan.
func LeadingInt(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, false
	}
	var n int64
	for _, ch := range s[:end] {
		n = n*10 + int64(ch-'0')
	}
	return n, true
}
