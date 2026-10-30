-- Koreksi koordinat "Gudang 1 Cimahi" pada database yang SUDAH berjalan.
-- (cmd/seed hanya membuat gudang baru; ia tidak mengubah baris yang sudah ada.)
-- Jalankan sekali:  psql -d stockrsd -f deploy/sql/2026-10-26-koreksi-koordinat-gudang-1.sql
BEGIN;

UPDATE gudangs
SET latitude  = -7.017319591938903,
    longitude = 107.648896219795,
    alamat    = 'Cimahi -7.017319591938903, 107.648896219795',
    updated_at = NOW()
WHERE nama = 'Gudang 1 Cimahi'
  AND deleted_at IS NULL;

-- Pastikan tepat 1 baris yang berubah; kalau 0, nama gudang di DB berbeda.
SELECT id, nama, latitude, longitude, alamat FROM gudangs WHERE nama = 'Gudang 1 Cimahi';

COMMIT;
