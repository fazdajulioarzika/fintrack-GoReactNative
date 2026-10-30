# FinTrack REST API v1

Base URL: `http://localhost:8080/api/v1`

## Konvensi

- Format JSON (UTF-8). Nominal uang adalah **integer rupiah** (tanpa desimal), maksimal 999.999.999.999.
- Autentikasi: header `Authorization: Bearer <access_token>`. Access token berlaku 15 menit; perbarui dengan `/auth/refresh`.
- Tanggal transaksi: RFC 3339 (`2026-10-29T12:30:00+07:00`). Filter tanggal: `YYYY-MM-DD` (dihitung menurut zona waktu user) atau RFC 3339. Bulan: `YYYY-MM`.
- Zona waktu user (`timezone`, default `Asia/Jakarta`) menentukan batas hari dan bulan pada dashboard, anggaran, dan analitik.
- `Idempotency-Key` (opsional, maks 100 karakter) pada `POST /transactions` dan `POST /savings-goals/{id}/contributions`. Request ulang dengan key yang sama mengembalikan hasil pertama (status `200`, bukan `201`).
- Format error: `{"error": {"code": "...", "message": "..."}}`

| HTTP | code                                                           | Arti                                |
| ---- | -------------------------------------------------------------- | ----------------------------------- |
| 400  | `validation_error`, `invalid_id`, `amount_too_large`           | Input tidak valid                   |
| 401  | `unauthorized`, `invalid_credentials`, `invalid_refresh_token` | Belum login / kredensial salah      |
| 404  | `not_found`                                                    | Data tidak ada atau milik user lain |
| 409  | `email_taken`, `budget_exists`                                 | Duplikat                            |
| 422  | `insufficient_savings`                                         | Penarikan melebihi tabungan         |
| 429  | `rate_limited`                                                 | Terlalu banyak percobaan            |

Rate limit: 10 request/menit per IP (burst 5) pada `/auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout`, dan `DELETE /auth/me`.

## Sistem

| Method | Path                          | Fungsi                               |
| ------ | ----------------------------- | ------------------------------------ |
| GET    | `/health` (di luar `/api/v1`) | Proses hidup                         |
| GET    | `/ready` (di luar `/api/v1`)  | Database terjangkau (503 jika tidak) |

## Autentikasi

| Method | Path             | Auth | Fungsi                                             |
| ------ | ---------------- | ---- | -------------------------------------------------- |
| POST   | `/auth/register` | -    | Buat akun → 201                                    |
| POST   | `/auth/login`    | -    | Login → 200                                        |
| POST   | `/auth/refresh`  | -    | Tukar refresh token (token lama langsung hangus)   |
| POST   | `/auth/logout`   | -    | Cabut refresh token → 204                          |
| GET    | `/auth/me`       | ✔    | Profil                                             |
| PATCH  | `/auth/me`       | ✔    | Ubah `name` dan/atau `timezone`                    |
| DELETE | `/auth/me`       | ✔    | Hapus akun + seluruh data (body: `password`) → 204 |

```json
POST /auth/register   {"name": "Budi", "email": "budi@example.com", "password": "rahasia123"}
→ 201 {"user": {"id": "...", "name": "Budi", "email": "...", "currency": "IDR", "timezone": "Asia/Jakarta"},
       "access_token": "...", "refresh_token": "...", "expires_in": 900}
```

Password 8 sampai 72 **byte**. Nama 2 sampai 100 karakter.

## Kategori

| Method | Path                               | Fungsi                                |
| ------ | ---------------------------------- | ------------------------------------- |
| GET    | `/categories?type=income\|expense` | Kategori sistem (8 expense, 6 income) |

## Transaksi

| Method | Path                 | Fungsi                                                                                                                                                           |
| ------ | -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| GET    | `/transactions`      | Daftar (query: `type`, `category_id`, `start_date`, `end_date`, `q`, `page`, `limit` ≤ 100, `sort` = `date_desc` \| `date_asc` \| `amount_desc` \| `amount_asc`) |
| POST   | `/transactions`      | Buat → 201                                                                                                                                                       |
| GET    | `/transactions/{id}` | Detail                                                                                                                                                           |
| PATCH  | `/transactions/{id}` | Ubah sebagian field                                                                                                                                              |
| DELETE | `/transactions/{id}` | Hapus → 204                                                                                                                                                      |

```json
POST /transactions
{"type": "expense", "amount": 25000, "category_id": "...", "transaction_date": "2026-10-29T12:30:00+07:00",
 "payment_method": "GoPay", "note": "Makan siang"}
```

Aturan: `amount` bilangan bulat > 0; kategori harus ada dan jenisnya sama dengan `type`. Respons daftar: `{"data": [...], "meta": {"page", "limit", "total", "total_pages"}}`.

## Dashboard (query `month=YYYY-MM`, default bulan berjalan)

| Method | Path                                 | Fungsi                                                                                     |
| ------ | ------------------------------------ | ------------------------------------------------------------------------------------------ |
| GET    | `/dashboard/summary`                 | `balance` (semua waktu), `month_income`, `month_expense`, `month_net`, `transaction_count` |
| GET    | `/dashboard/cashflow`                | Satu titik per hari (hari kosong bernilai 0)                                               |
| GET    | `/dashboard/categories?type=expense` | Distribusi per kategori + persen                                                           |

Saldo **selalu dihitung dari transaksi**, tidak pernah disimpan sebagai angka terpisah.

## Anggaran

| Method | Path                     | Fungsi                                                                                                      |
| ------ | ------------------------ | ----------------------------------------------------------------------------------------------------------- |
| GET    | `/budgets?month=YYYY-MM` | Daftar + pemakaian                                                                                          |
| POST   | `/budgets`               | `{"category_id", "period": "YYYY-MM", "limit_amount"}`, hanya kategori expense, satu per kategori per bulan |
| PATCH  | `/budgets/{id}`          | `{"limit_amount"}`                                                                                          |
| DELETE | `/budgets/{id}`          | Hapus → 204                                                                                                 |

Status: `normal` (< 80%), `warning` (80% sampai < 100%), `exceeded` (≥ 100%). Hanya pengeluaran yang dihitung.

## Target tabungan

| Method | Path                                     | Fungsi                                                                           |
| ------ | ---------------------------------------- | -------------------------------------------------------------------------------- |
| GET    | `/savings-goals?status=active\|achieved` | Daftar + total                                                                   |
| POST   | `/savings-goals`                         | `{"name", "target_amount", "target_date"?, "icon"?}`                             |
| GET    | `/savings-goals/{id}`                    | Detail                                                                           |
| PATCH  | `/savings-goals/{id}`                    | Ubah; `clear_target_date: true` menghapus tanggal                                |
| DELETE | `/savings-goals/{id}`                    | Hapus target + riwayat (atomik) → 204                                            |
| GET    | `/savings-goals/{id}/contributions`      | Riwayat (`page`, `limit`)                                                        |
| POST   | `/savings-goals/{id}/contributions`      | `{"type": "deposit"\|"withdraw", "amount", "note"?, "contribution_date"?}` → 201 |

`progress_percent` dibatasi 100. Status otomatis `achieved` saat terkumpul ≥ target dan kembali `active` jika turun. Penarikan melebihi saldo target ditolak (422). Kontribusi hanya pencatatan alokasi dan **tidak** mengubah saldo transaksi.

## Analitik (query `range=this_month|last_month|custom`, plus `start_date` dan `end_date` untuk `custom`, maks 366 hari)

| Method | Path                                      | Fungsi                                                                         |
| ------ | ----------------------------------------- | ------------------------------------------------------------------------------ |
| GET    | `/analytics/summary`                      | Total pemasukan/pengeluaran, selisih, distribusi kategori                      |
| GET    | `/analytics/trend?granularity=day\|month` | Titik tren                                                                     |
| GET    | `/analytics/compare`                      | Periode ini vs periode sebelumnya; `percent` bernilai `null` jika pembanding 0 |

## Batasan yang diketahui

- Access token tetap berlaku sampai kedaluwarsa (maks 15 menit) walau akun sudah dihapus atau logout.
- Rate limiter tersimpan di memori satu server (cukup untuk satu instance).
- Belum ada endpoint ganti password dan kategori buatan sendiri (pasca-MVP di PRD).
