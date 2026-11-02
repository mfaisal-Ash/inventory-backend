-- Tanggal catat barang (Kelola Barang) + stok awal/keluar awal untuk neraca.
--
-- GORM AutoMigrate menambah kolom baru di tabel `barang` secara otomatis saat
-- backend baru dijalankan:
--   tanggal             date    NOT NULL DEFAULT CURRENT_DATE
--   stok_awal           integer NOT NULL DEFAULT 0
--   keluar_awal         integer NOT NULL DEFAULT 0
--   stok_awal_gudang_id bigint  NULL
-- Akibatnya SEMUA barang lama ikut diberi tanggal = hari migrasi berjalan.
-- Skrip ini mengembalikan tanggal barang lama ke tanggal pembuatannya supaya
-- periode (Neraca & Analisis Data) tidak bergeser ke hari deploy.
--
-- URUTAN: 1) deploy backend baru (jalankan sekali agar AutoMigrate jalan)
--         2) BACKUP database
--         3) psql -d stockrsd -f deploy/sql/2026-10-31-tanggal-catat-barang.sql
--
-- Aman dijalankan ulang (hanya menyentuh barang yang belum punya stok awal
-- tercatat DAN tanggalnya masih sama dengan hari ini). Zona waktu: WIB.
BEGIN;

UPDATE barang
SET    tanggal = (created_at AT TIME ZONE 'Asia/Jakarta')::date
WHERE  stok_awal = 0
  AND  keluar_awal = 0
  AND  tanggal = (now() AT TIME ZONE 'Asia/Jakarta')::date
  AND  (created_at AT TIME ZONE 'Asia/Jakarta')::date < (now() AT TIME ZONE 'Asia/Jakarta')::date;

-- Barang lama TIDAK diberi stok_awal: stok awal sebelum fitur ini tidak
-- tercatat per tanggal, jadi tetap muncul sebagai "selisih tak terlacak" di
-- Analisis Data sampai dikoreksi lewat Stock Opname.

COMMIT;
