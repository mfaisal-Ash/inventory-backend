// Package docnumber menghasilkan nomor dokumen berurutan per bulan (format
// "<prefix>-<YYYYMM>-<4 digit>", mis. "BM-202609-0001") untuk dokumen
// seperti Barang Masuk, Barang Keluar, Pengajuan Barang, dan Stock Opname.
//
// Skema lama menempelkan nomor ke time.Now().UnixNano() % 100000 — hasilnya
// kelihatan acak buat pengguna (tidak berurutan, tidak bisa ditebak dokumen
// keberapa ini dalam sebulan) dan secara teori dua request yang persis
// bersamaan di nanodetik yang sama bisa menghasilkan nomor yang sama.
// Package ini menggantinya dengan counter berurutan yang disimpan di tabel
// document_counters dan di-increment secara ATOMIK lewat upsert Postgres
// (INSERT ... ON CONFLICT DO UPDATE ... RETURNING), jadi aman dipakai
// bersamaan oleh banyak request tanpa nomor bentrok/lompat, tanpa perlu
// row-locking manual di pemanggilnya.
package docnumber

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Next menghasilkan nomor dokumen berikutnya untuk prefix ini di periode
// (bulan) berjalan. Pass `db` sebagai koneksi biasa, atau sebagai `tx` di
// dalam transaksi yang sedang berjalan (mis. saat Pengajuan Barang otomatis
// dikonversi jadi dokumen Barang Masuk/Keluar) supaya nomor ini ikut
// ter-rollback kalau transaksi induknya gagal — persis pola FOR UPDATE yang
// sudah dipakai di tempat lain di proyek ini untuk operasi yang harus atomik.
func Next(db *gorm.DB, prefix string) (string, error) {
	period := time.Now().Format("200601")
	key := prefix + "-" + period

	var counter int
	if err := db.Raw(
		`INSERT INTO document_counters (key, counter, updated_at)
		 VALUES (?, 1, now())
		 ON CONFLICT (key) DO UPDATE SET counter = document_counters.counter + 1, updated_at = now()
		 RETURNING counter`,
		key,
	).Scan(&counter).Error; err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s-%04d", prefix, period, counter), nil
}
