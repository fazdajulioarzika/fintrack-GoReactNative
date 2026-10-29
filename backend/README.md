# FinTrack Backend (Phase 1: Autentikasi)

## Menjalankan
```bash
cp .env.example .env        # lalu isi MONGO_URI dan JWT_SECRET
go mod tidy                 # mengunduh dependensi
go run ./cmd/api
```

## Endpoint
| Method | Path | Auth | Fungsi |
|---|---|---|---|
| GET | /health | - | Cek server |
| POST | /api/v1/auth/register | - | Buat akun |
| POST | /api/v1/auth/login | - | Login |
| POST | /api/v1/auth/refresh | - | Tukar refresh token |
| POST | /api/v1/auth/logout | - | Cabut refresh token |
| GET | /api/v1/auth/me | Bearer | Profil |
| PATCH | /api/v1/auth/me | Bearer | Ubah nama / zona waktu |
| DELETE | /api/v1/auth/me | Bearer | Hapus akun (perlu password) |

## Tes cepat dengan curl
```bash
curl -X POST localhost:8080/api/v1/auth/register -H "Content-Type: application/json" \
  -d '{"name":"Budi","email":"budi@example.com","password":"rahasia123"}'

curl localhost:8080/api/v1/auth/me -H "Authorization: Bearer ACCESS_TOKEN"
```
