-- Menjamin counter nomor dokumen tersedia pada deployment lama.
-- Aman dijalankan berulang kali.
CREATE TABLE IF NOT EXISTS document_counters (
    key varchar(32) PRIMARY KEY,
    counter integer NOT NULL DEFAULT 0,
    updated_at timestamptz NOT NULL DEFAULT now()
);
