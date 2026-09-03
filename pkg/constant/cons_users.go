package constant

const (
	ErrUsersUserNotFound = "user tidak ditemukan"
	ErrUsernameDuplikat  = "username sudah digunakan"
	ErrPasswordLamaSalah = "password lama salah"

	ErrGagalMengambilDaftarUser = "gagal mengambil daftar user"
	ErrGagalMembuatUser         = "gagal membuat user"
	ErrGagalMemperbaruiUser     = "gagal memperbarui user"
	ErrGagalMengubahPassword    = "gagal mengubah password"

	MsgDaftarUserBerhasil   = "daftar user berhasil diambil"
	MsgDetailUserBerhasil   = "detail user berhasil diambil"
	MsgUserBerhasilDibuat   = "user berhasil dibuat"
	MsgUserBerhasilDiubah   = "user berhasil diperbarui"
	MsgPasswordBerhasilUbah = "password berhasil diubah"
)

const QueryIDEq = "id = ?"
