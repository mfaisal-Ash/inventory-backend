-- Pengajuan Barang: izinkan item input MANUAL (barang belum terdaftar di Kelola Barang).
--
-- Sebelumnya kolom pengajuan_barang_items.barang_id NOT NULL + foreign key ke
-- tabel barang, sedangkan item manual tidak punya barang_id (terkirim 0) ->
-- INSERT gagal (HTTP 500 "gagal membuat pengajuan barang").
--
-- GORM AutoMigrate biasanya melepas NOT NULL sendiri saat backend baru jalan;
-- skrip ini menjaminnya (aman dijalankan berulang).
--   psql -d stockrsd -f deploy/sql/2026-11-07-pengajuan-item-barang-id-nullable.sql
BEGIN;
ALTER TABLE pengajuan_barang_items ALTER COLUMN barang_id DROP NOT NULL;
COMMIT;
