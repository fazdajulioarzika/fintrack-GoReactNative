package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "time/tzdata"

	"fintrack-backend/internal/auth"
	"fintrack-backend/internal/config"
	"fintrack-backend/internal/database"
	"fintrack-backend/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const jwtSecret = "test-secret-test-secret-test-secret-1234"

var (
	router http.Handler
	db     *mongo.Database
)

func TestMain(m *testing.M) {
	uri := os.Getenv("TEST_MONGO_URI")
	if uri == "" {
		fmt.Println("TEST_MONGO_URI belum diset: tes integrasi dilewati")
		os.Exit(0)
	}
	ctx := context.Background()
	client, d, err := database.Connect(ctx, uri, fmt.Sprintf("fintrack_test_%d", time.Now().UnixNano()))
	if err != nil {
		fmt.Println("koneksi MongoDB gagal:", err)
		os.Exit(1)
	}
	db = d
	if err := database.Setup(ctx, db); err != nil {
		fmt.Println("setup database gagal:", err)
		os.Exit(1)
	}

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	router = server.NewRouter(&config.Config{
		Env:            "test",
		JWTSecret:      jwtSecret,
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     24 * time.Hour,
		AuthRatePerMin: 6000, // rate limit dilonggarkan agar tes tidak terkena 429
		AuthBurst:      1000,
	}, client, db)

	code := m.Run()
	_ = db.Drop(ctx)
	_ = client.Disconnect(ctx)
	os.Exit(code)
}

// ---------- Helper ----------

type resp struct {
	Status int
	Body   map[string]any
}

func call(t *testing.T, method, path, token string, body any, hdr ...string) resp {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, "/api/v1"+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	out := resp{Status: w.Code}
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &out.Body)
	}
	return out
}

func wantStatus(t *testing.T, r resp, want int) {
	t.Helper()
	if r.Status != want {
		t.Fatalf("status = %d, want %d (body: %v)", r.Status, want, r.Body)
	}
}

func dig(m map[string]any, keys ...string) any {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[k]
	}
	return cur
}

func str(m map[string]any, keys ...string) string {
	s, _ := dig(m, keys...).(string)
	return s
}

func num(m map[string]any, keys ...string) int64 {
	f, _ := dig(m, keys...).(float64)
	return int64(f)
}

type testUser struct{ ID, Email, Password, Access, Refresh string }

var userSeq atomic.Int64

func newUser(t *testing.T) testUser {
	t.Helper()
	u := testUser{
		Email:    fmt.Sprintf("user%d-%d@example.com", time.Now().UnixNano(), userSeq.Add(1)),
		Password: "rahasia123",
	}
	r := call(t, "POST", "/auth/register", "", map[string]any{"name": "Tester", "email": u.Email, "password": u.Password})
	wantStatus(t, r, 201)
	u.ID = str(r.Body, "user", "id")
	u.Access = str(r.Body, "access_token")
	u.Refresh = str(r.Body, "refresh_token")
	return u
}

func categoryID(t *testing.T, token, txType, name string) string {
	t.Helper()
	r := call(t, "GET", "/categories?type="+txType, token, nil)
	wantStatus(t, r, 200)
	list, _ := r.Body["data"].([]any)
	for _, it := range list {
		if m, ok := it.(map[string]any); ok && m["name"] == name {
			return m["id"].(string)
		}
	}
	t.Fatalf("kategori %q (%s) tidak ditemukan", name, txType)
	return ""
}

func mkTx(t *testing.T, token, typ string, amount int64, cat string) string {
	t.Helper()
	r := call(t, "POST", "/transactions", token, map[string]any{
		"type": typ, "amount": amount, "category_id": cat,
		"transaction_date": time.Now().Format(time.RFC3339),
	})
	wantStatus(t, r, 201)
	return str(r.Body, "transaction", "id")
}

func thisMonth() string {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	return time.Now().In(loc).Format("2006-01")
}

func mkGoal(t *testing.T, token string, target int64) string {
	t.Helper()
	r := call(t, "POST", "/savings-goals", token, map[string]any{"name": "Dana darurat", "target_amount": target})
	wantStatus(t, r, 201)
	return str(r.Body, "goal", "id")
}

// ---------- Autentikasi ----------

func TestAuthFlow(t *testing.T) {
	email := fmt.Sprintf("auth%d@example.com", time.Now().UnixNano())
	pw := "rahasia123"

	reg := call(t, "POST", "/auth/register", "", map[string]any{"name": "Budi", "email": email, "password": pw})
	wantStatus(t, reg, 201)
	if str(reg.Body, "access_token") == "" || str(reg.Body, "refresh_token") == "" {
		t.Fatal("token kosong pada respons register")
	}
	if dig(reg.Body, "user", "password_hash") != nil {
		t.Fatal("password_hash bocor ke respons")
	}

	// Password harus tersimpan sebagai hash bcrypt, bukan plaintext
	var doc struct {
		PasswordHash string `bson:"password_hash"`
	}
	if err := db.Collection("users").FindOne(context.Background(), bson.M{"email": email}).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if doc.PasswordHash == pw || !strings.HasPrefix(doc.PasswordHash, "$2") {
		t.Fatalf("password tidak tersimpan sebagai hash: %q", doc.PasswordHash)
	}

	// Email sama (termasuk beda huruf besar) ditolak
	dup := call(t, "POST", "/auth/register", "", map[string]any{"name": "Budi", "email": strings.ToUpper(email), "password": pw})
	wantStatus(t, dup, 409)

	invalid := map[string]map[string]any{
		"password pendek":          {"name": "Budi", "email": "a1@example.com", "password": "abc"},
		"email tidak valid":        {"name": "Budi", "email": "bukan-email", "password": pw},
		"nama 1 huruf":             {"name": "B", "email": "a2@example.com", "password": pw},
		"password > 72 byte":       {"name": "Budi", "email": "a3@example.com", "password": strings.Repeat("é", 40)},
		"field tidak lengkap":      {"name": "Budi"},
		"email berupa objek NoSQL": {"name": "Budi", "email": map[string]any{"$ne": ""}, "password": pw},
	}
	for name, body := range invalid {
		if r := call(t, "POST", "/auth/register", "", body); r.Status != 400 {
			t.Errorf("register %s: status = %d, want 400", name, r.Status)
		}
	}

	// Login
	login := call(t, "POST", "/auth/login", "", map[string]any{"email": email, "password": pw})
	wantStatus(t, login, 200)

	wrong := call(t, "POST", "/auth/login", "", map[string]any{"email": email, "password": "salah12345"})
	wantStatus(t, wrong, 401)
	unknown := call(t, "POST", "/auth/login", "", map[string]any{"email": "tidakada@example.com", "password": "salah12345"})
	wantStatus(t, unknown, 401)
	if str(wrong.Body, "error", "message") != str(unknown.Body, "error", "message") {
		t.Error("pesan error login harus sama agar email terdaftar tidak bisa ditebak")
	}
	inject := call(t, "POST", "/auth/login", "", map[string]any{"email": map[string]any{"$ne": ""}, "password": map[string]any{"$ne": ""}})
	wantStatus(t, inject, 400)

	// Profil
	wantStatus(t, call(t, "GET", "/auth/me", "", nil), 401)
	me := call(t, "GET", "/auth/me", str(login.Body, "access_token"), nil)
	wantStatus(t, me, 200)
	if str(me.Body, "user", "email") != email {
		t.Error("email profil tidak sesuai")
	}

	// Refresh token dirotasi: token lama langsung tidak berlaku
	oldRefresh := str(login.Body, "refresh_token")
	ref := call(t, "POST", "/auth/refresh", "", map[string]any{"refresh_token": oldRefresh})
	wantStatus(t, ref, 200)
	newRefresh := str(ref.Body, "refresh_token")
	if newRefresh == "" || newRefresh == oldRefresh {
		t.Fatal("refresh token harus berganti")
	}
	wantStatus(t, call(t, "POST", "/auth/refresh", "", map[string]any{"refresh_token": oldRefresh}), 401)
	wantStatus(t, call(t, "POST", "/auth/refresh", "", map[string]any{"refresh_token": "ngawur"}), 401)

	// Logout mencabut sesi
	wantStatus(t, call(t, "POST", "/auth/logout", "", map[string]any{"refresh_token": newRefresh}), 204)
	wantStatus(t, call(t, "POST", "/auth/refresh", "", map[string]any{"refresh_token": newRefresh}), 401)
}

// ---------- Keamanan token ----------

func TestTokenSecurity(t *testing.T) {
	a := newUser(t)
	b := newUser(t)
	probe := func(name, token string, want int) {
		t.Helper()
		if r := call(t, "GET", "/auth/me", token, nil); r.Status != want {
			t.Errorf("%s: status = %d, want %d", name, r.Status, want)
		}
	}

	probe("token valid", a.Access, 200)

	parts := strings.Split(a.Access, ".")
	partsB := strings.Split(b.Access, ".")
	probe("tanda tangan diganti", parts[0]+"."+parts[1]+"."+strings.Repeat("A", len(parts[2])), 401)
	probe("payload user lain + tanda tangan A", parts[0]+"."+partsB[1]+"."+parts[2], 401)

	expired, _ := auth.NewAccessToken(a.ID, jwtSecret, -time.Minute)
	probe("kedaluwarsa", expired, 401)
	otherSecret, _ := auth.NewAccessToken(a.ID, "secret-lain-secret-lain-secret-lain-123", time.Hour)
	probe("secret berbeda", otherSecret, 401)
	badSubject, _ := auth.NewAccessToken("bukan-object-id", jwtSecret, time.Hour)
	probe("subject bukan ObjectID", badSubject, 401)

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   a.ID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	probe("algoritma none", none, 401)

	probe("refresh token dipakai sebagai access token", a.Refresh, 401)
	probe("string acak", "abc", 401)

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Basic xyz")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Errorf("skema Basic: status = %d, want 401", w.Code)
	}
}

func TestPrivateEndpointsRequireAuth(t *testing.T) {
	zero := "000000000000000000000000"
	endpoints := [][2]string{
		{"GET", "/auth/me"}, {"PATCH", "/auth/me"}, {"DELETE", "/auth/me"},
		{"GET", "/categories"},
		{"GET", "/transactions"}, {"POST", "/transactions"},
		{"GET", "/transactions/" + zero}, {"PATCH", "/transactions/" + zero}, {"DELETE", "/transactions/" + zero},
		{"GET", "/dashboard/summary"}, {"GET", "/dashboard/cashflow"}, {"GET", "/dashboard/categories"},
		{"GET", "/budgets"}, {"POST", "/budgets"}, {"PATCH", "/budgets/" + zero}, {"DELETE", "/budgets/" + zero},
		{"GET", "/savings-goals"}, {"POST", "/savings-goals"},
		{"GET", "/savings-goals/" + zero}, {"PATCH", "/savings-goals/" + zero}, {"DELETE", "/savings-goals/" + zero},
		{"GET", "/savings-goals/" + zero + "/contributions"}, {"POST", "/savings-goals/" + zero + "/contributions"},
		{"GET", "/analytics/summary"}, {"GET", "/analytics/trend"}, {"GET", "/analytics/compare"},
	}
	for _, e := range endpoints {
		if r := call(t, e[0], e[1], "", nil); r.Status != 401 {
			t.Errorf("%s %s tanpa token: status = %d, want 401", e[0], e[1], r.Status)
		}
	}
}

// ---------- Kepemilikan data ----------

func TestOwnershipIsolation(t *testing.T) {
	a := newUser(t)
	b := newUser(t)

	expense := categoryID(t, a.Access, "expense", "Makanan dan minuman")
	txID := mkTx(t, a.Access, "expense", 25000, expense)

	bud := call(t, "POST", "/budgets", a.Access, map[string]any{"category_id": expense, "period": thisMonth(), "limit_amount": 100000})
	wantStatus(t, bud, 201)
	budID := str(bud.Body, "budget", "id")

	goalID := mkGoal(t, a.Access, 1000000)

	denied := []struct {
		name, method, path string
		body               any
	}{
		{"baca transaksi", "GET", "/transactions/" + txID, nil},
		{"ubah transaksi", "PATCH", "/transactions/" + txID, map[string]any{"amount": 1}},
		{"hapus transaksi", "DELETE", "/transactions/" + txID, nil},
		{"ubah anggaran", "PATCH", "/budgets/" + budID, map[string]any{"limit_amount": 1}},
		{"hapus anggaran", "DELETE", "/budgets/" + budID, nil},
		{"baca target", "GET", "/savings-goals/" + goalID, nil},
		{"ubah target", "PATCH", "/savings-goals/" + goalID, map[string]any{"name": "dibajak"}},
		{"setor ke target", "POST", "/savings-goals/" + goalID + "/contributions", map[string]any{"type": "deposit", "amount": 1000}},
		{"riwayat target", "GET", "/savings-goals/" + goalID + "/contributions", nil},
		{"hapus target", "DELETE", "/savings-goals/" + goalID, nil},
	}
	for _, d := range denied {
		if r := call(t, d.method, d.path, b.Access, d.body); r.Status != 404 {
			t.Errorf("%s oleh user lain: status = %d, want 404", d.name, r.Status)
		}
	}

	// Daftar milik B harus kosong
	list := call(t, "GET", "/transactions", b.Access, nil)
	wantStatus(t, list, 200)
	if num(list.Body, "meta", "total") != 0 {
		t.Error("daftar transaksi B tidak boleh berisi data A")
	}
	if bl := call(t, "GET", "/budgets?month="+thisMonth(), b.Access, nil); len(bl.Body["data"].([]any)) != 0 {
		t.Error("daftar anggaran B tidak boleh berisi data A")
	}

	// Mengirim user_id milik A di body tidak boleh berpengaruh
	r := call(t, "POST", "/transactions", b.Access, map[string]any{
		"type": "expense", "amount": 5000, "category_id": expense,
		"transaction_date": time.Now().Format(time.RFC3339), "user_id": a.ID,
	})
	wantStatus(t, r, 201)
	if num(call(t, "GET", "/transactions", a.Access, nil).Body, "meta", "total") != 1 {
		t.Error("transaksi yang dikirim B tidak boleh masuk ke akun A")
	}

	// Data A tetap utuh setelah semua percobaan B
	wantStatus(t, call(t, "GET", "/transactions/"+txID, a.Access, nil), 200)
	if num(call(t, "GET", "/savings-goals/"+goalID, a.Access, nil).Body, "goal", "saved_amount") != 0 {
		t.Error("tabungan A berubah")
	}
}

// ---------- Transaksi ----------

func TestTransactionValidation(t *testing.T) {
	u := newUser(t)
	expense := categoryID(t, u.Access, "expense", "Transportasi")
	now := time.Now().Format(time.RFC3339)

	base := func() map[string]any {
		return map[string]any{"type": "expense", "amount": 10000, "category_id": expense, "transaction_date": now}
	}
	with := func(k string, v any) map[string]any { m := base(); m[k] = v; return m }
	without := func(k string) map[string]any { m := base(); delete(m, k); return m }

	cases := map[string]map[string]any{
		"nominal nol":              with("amount", 0),
		"nominal negatif":          with("amount", -5),
		"nominal desimal":          with("amount", 1.5),
		"nominal melebihi batas":   with("amount", 1000000000000),
		"nominal berupa teks":      with("amount", "10000"),
		"jenis tidak dikenal":      with("type", "transfer"),
		"kategori tidak ada":       without("category_id"),
		"kategori bukan ObjectID":  with("category_id", "abc"),
		"kategori tidak ditemukan": with("category_id", "000000000000000000000000"),
		"kategori beda jenis":      with("type", "income"),
		"tanggal tidak ada":        without("transaction_date"),
		"tanggal format salah":     with("transaction_date", "kemarin"),
		"catatan terlalu panjang":  with("note", strings.Repeat("x", 501)),
	}
	for name, body := range cases {
		if r := call(t, "POST", "/transactions", u.Access, body); r.Status != 400 {
			t.Errorf("%s: status = %d, want 400 (body: %v)", name, r.Status, r.Body)
		}
	}
	wantStatus(t, call(t, "POST", "/transactions", u.Access, base()), 201)

	// Ubah jenis ke income dengan kategori expense harus ditolak
	id := mkTx(t, u.Access, "expense", 1000, expense)
	wantStatus(t, call(t, "PATCH", "/transactions/"+id, u.Access, map[string]any{"type": "income"}), 400)
	wantStatus(t, call(t, "PATCH", "/transactions/"+id, u.Access, map[string]any{"amount": 0}), 400)
	wantStatus(t, call(t, "GET", "/transactions/bukan-id", u.Access, nil), 400)
}

func TestTransactionIdempotency(t *testing.T) {
	a := newUser(t)
	b := newUser(t)
	catA := categoryID(t, a.Access, "expense", "Belanja")
	catB := categoryID(t, b.Access, "expense", "Belanja")
	body := func(cat string) map[string]any {
		return map[string]any{"type": "expense", "amount": 50000, "category_id": cat, "transaction_date": time.Now().Format(time.RFC3339)}
	}

	first := call(t, "POST", "/transactions", a.Access, body(catA), "Idempotency-Key", "abc-123")
	wantStatus(t, first, 201)
	second := call(t, "POST", "/transactions", a.Access, body(catA), "Idempotency-Key", "abc-123")
	wantStatus(t, second, 200)
	if str(first.Body, "transaction", "id") != str(second.Body, "transaction", "id") {
		t.Error("request ulang harus mengembalikan transaksi yang sama")
	}
	if num(call(t, "GET", "/transactions", a.Access, nil).Body, "meta", "total") != 1 {
		t.Error("hanya boleh ada satu transaksi")
	}

	// Key yang sama milik user lain tidak saling mengganggu
	wantStatus(t, call(t, "POST", "/transactions", b.Access, body(catB), "Idempotency-Key", "abc-123"), 201)
}

func TestBalanceStaysConsistent(t *testing.T) {
	u := newUser(t)
	salary := categoryID(t, u.Access, "income", "Gaji")
	food := categoryID(t, u.Access, "expense", "Makanan dan minuman")

	check := func(balance, income, expense int64) {
		t.Helper()
		r := call(t, "GET", "/dashboard/summary", u.Access, nil)
		wantStatus(t, r, 200)
		if got := num(r.Body, "balance"); got != balance {
			t.Fatalf("balance = %d, want %d", got, balance)
		}
		if got := num(r.Body, "month_income"); got != income {
			t.Fatalf("month_income = %d, want %d", got, income)
		}
		if got := num(r.Body, "month_expense"); got != expense {
			t.Fatalf("month_expense = %d, want %d", got, expense)
		}
	}

	check(0, 0, 0)
	mkTx(t, u.Access, "income", 5000000, salary)
	expID := mkTx(t, u.Access, "expense", 75000, food)
	check(4925000, 5000000, 75000)

	wantStatus(t, call(t, "PATCH", "/transactions/"+expID, u.Access, map[string]any{"amount": 100000}), 200)
	check(4900000, 5000000, 100000)

	cats := call(t, "GET", "/dashboard/categories?type=expense", u.Access, nil)
	wantStatus(t, cats, 200)
	if num(cats.Body, "total") != 100000 {
		t.Error("total distribusi kategori harus sama dengan total pengeluaran")
	}

	wantStatus(t, call(t, "DELETE", "/transactions/"+expID, u.Access, nil), 204)
	check(5000000, 5000000, 0)
}

func TestListAndDashboardFilters(t *testing.T) {
	u := newUser(t)
	food := categoryID(t, u.Access, "expense", "Makanan dan minuman")
	r := call(t, "POST", "/transactions", u.Access, map[string]any{
		"type": "expense", "amount": 20000, "category_id": food, "note": "makan siang",
		"transaction_date": time.Now().Format(time.RFC3339),
	})
	wantStatus(t, r, 201)

	if got := num(call(t, "GET", "/transactions?q=makan", u.Access, nil).Body, "meta", "total"); got != 1 {
		t.Errorf("pencarian 'makan': total = %d, want 1", got)
	}
	// Input pencarian diperlakukan sebagai teks biasa, bukan pola regex
	if got := num(call(t, "GET", "/transactions?q=.*", u.Access, nil).Body, "meta", "total"); got != 0 {
		t.Errorf("pencarian '.*' tidak boleh cocok dengan semuanya, total = %d", got)
	}

	clamp := call(t, "GET", "/transactions?limit=1000&page=0", u.Access, nil)
	wantStatus(t, clamp, 200)
	if num(clamp.Body, "meta", "limit") != 100 || num(clamp.Body, "meta", "page") != 1 {
		t.Errorf("limit dan page harus dibatasi: %v", clamp.Body["meta"])
	}

	for _, path := range []string{
		"/transactions?type=transfer",
		"/transactions?category_id=zzz",
		"/transactions?start_date=kemarin",
		"/dashboard/summary?month=abc",
		"/dashboard/cashflow?month=2026-13",
		"/dashboard/categories?month=abc",
		"/dashboard/categories?type=foo",
		"/categories?type=foo",
	} {
		if r := call(t, "GET", path, u.Access, nil); r.Status != 400 {
			t.Errorf("GET %s: status = %d, want 400", path, r.Status)
		}
	}
}

// ---------- Anggaran ----------

func TestBudgetUsageAndStatus(t *testing.T) {
	u := newUser(t)
	food := categoryID(t, u.Access, "expense", "Makanan dan minuman")
	salary := categoryID(t, u.Access, "income", "Gaji")
	month := thisMonth()

	created := call(t, "POST", "/budgets", u.Access, map[string]any{"category_id": food, "period": month, "limit_amount": 100000})
	wantStatus(t, created, 201)
	budID := str(created.Body, "budget", "id")

	budget := func() (status string, spent int64) {
		t.Helper()
		r := call(t, "GET", "/budgets?month="+month, u.Access, nil)
		wantStatus(t, r, 200)
		for _, it := range r.Body["data"].([]any) {
			m := it.(map[string]any)
			if m["id"] == budID {
				return m["status"].(string), int64(m["spent_amount"].(float64))
			}
		}
		t.Fatal("anggaran tidak ditemukan")
		return
	}
	expect := func(wantStatusStr string, wantSpent int64) {
		t.Helper()
		if s, sp := budget(); s != wantStatusStr || sp != wantSpent {
			t.Fatalf("anggaran = (%s, %d), want (%s, %d)", s, sp, wantStatusStr, wantSpent)
		}
	}

	expect("normal", 0)
	mkTx(t, u.Access, "expense", 79999, food)
	expect("normal", 79999) // 79,999% masih normal
	mkTx(t, u.Access, "expense", 1, food)
	expect("warning", 80000) // tepat 80%
	over := mkTx(t, u.Access, "expense", 20000, food)
	expect("exceeded", 100000) // tepat 100%

	mkTx(t, u.Access, "income", 1000000, salary) // pemasukan tidak mengurangi anggaran
	expect("exceeded", 100000)

	wantStatus(t, call(t, "DELETE", "/transactions/"+over, u.Access, nil), 204) // hapus transaksi memperbarui anggaran
	expect("warning", 80000)

	// Aturan pembuatan
	wantStatus(t, call(t, "POST", "/budgets", u.Access, map[string]any{"category_id": food, "period": month, "limit_amount": 5000}), 409)
	wantStatus(t, call(t, "POST", "/budgets", u.Access, map[string]any{"category_id": salary, "period": month, "limit_amount": 5000}), 400)
	wantStatus(t, call(t, "POST", "/budgets", u.Access, map[string]any{"category_id": food, "period": "2026-13", "limit_amount": 5000}), 400)
	wantStatus(t, call(t, "POST", "/budgets", u.Access, map[string]any{"category_id": food, "period": "2025-01", "limit_amount": 0}), 400)
	wantStatus(t, call(t, "POST", "/budgets", u.Access, map[string]any{"category_id": food, "period": "2025-01", "limit_amount": 1.5}), 400)

	wantStatus(t, call(t, "PATCH", "/budgets/"+budID, u.Access, map[string]any{"limit_amount": 500000}), 200)
	expect("normal", 80000)
	wantStatus(t, call(t, "DELETE", "/budgets/"+budID, u.Access, nil), 204)
}

// ---------- Target tabungan ----------

func TestSavingsFlow(t *testing.T) {
	u := newUser(t)
	id := mkGoal(t, u.Access, 1000000)
	path := "/savings-goals/" + id + "/contributions"

	contribute := func(typ string, amount int64, hdr ...string) resp {
		return call(t, "POST", path, u.Access, map[string]any{"type": typ, "amount": amount}, hdr...)
	}
	expectGoal := func(saved int64, status string) {
		t.Helper()
		r := call(t, "GET", "/savings-goals/"+id, u.Access, nil)
		wantStatus(t, r, 200)
		if got := num(r.Body, "goal", "saved_amount"); got != saved {
			t.Fatalf("saved_amount = %d, want %d", got, saved)
		}
		if got := str(r.Body, "goal", "status"); got != status {
			t.Fatalf("status = %s, want %s", got, status)
		}
	}

	expectGoal(0, "active")

	r := contribute("deposit", 400000)
	wantStatus(t, r, 201)
	if num(r.Body, "goal", "progress_percent") != 40 {
		t.Errorf("progres = %v, want 40", dig(r.Body, "goal", "progress_percent"))
	}

	wantStatus(t, contribute("deposit", 600000), 201)
	expectGoal(1000000, "achieved")

	wantStatus(t, contribute("withdraw", 200000), 201)
	expectGoal(800000, "active") // turun di bawah target: kembali aktif

	over := contribute("withdraw", 900000)
	wantStatus(t, over, 422)
	if str(over.Body, "error", "code") != "insufficient_savings" {
		t.Errorf("kode error = %s", str(over.Body, "error", "code"))
	}
	expectGoal(800000, "active") // penarikan gagal tidak mengubah saldo

	// Anti-duplikasi
	wantStatus(t, contribute("deposit", 50000, "Idempotency-Key", "dep-1"), 201)
	wantStatus(t, contribute("deposit", 50000, "Idempotency-Key", "dep-1"), 200)
	expectGoal(850000, "active")

	// Hanya kontribusi yang berhasil tercatat (4), yang gagal tidak
	list := call(t, "GET", path, u.Access, nil)
	wantStatus(t, list, 200)
	if got := num(list.Body, "meta", "total"); got != 4 {
		t.Errorf("riwayat = %d, want 4", got)
	}

	wantStatus(t, contribute("transfer", 1000), 400)
	wantStatus(t, contribute("deposit", 0), 400)
	wantStatus(t, contribute("deposit", 1000000000000), 400)

	// Mengecilkan target mengubah status
	wantStatus(t, call(t, "PATCH", "/savings-goals/"+id, u.Access, map[string]any{"target_amount": 500000}), 200)
	expectGoal(850000, "achieved")
	if p := num(call(t, "GET", "/savings-goals/"+id, u.Access, nil).Body, "goal", "progress_percent"); p != 100 {
		t.Errorf("progres tidak boleh melebihi 100, dapat %d", p)
	}

	// Hapus target ikut menghapus riwayatnya
	wantStatus(t, call(t, "DELETE", "/savings-goals/"+id, u.Access, nil), 204)
	wantStatus(t, call(t, "GET", "/savings-goals/"+id, u.Access, nil), 404)
	oid, _ := bson.ObjectIDFromHex(id)
	n, err := db.Collection("savings_contributions").CountDocuments(context.Background(), bson.M{"goal_id": oid})
	if err != nil || n != 0 {
		t.Errorf("riwayat kontribusi masih ada: n=%d err=%v", n, err)
	}
}

// Penarikan bersamaan tidak boleh membuat saldo negatif atau melewati jumlah yang tersedia.
func TestSavingsConcurrentWithdrawals(t *testing.T) {
	u := newUser(t)
	id := mkGoal(t, u.Access, 5000000)
	path := "/savings-goals/" + id + "/contributions"
	wantStatus(t, call(t, "POST", path, u.Access, map[string]any{"type": "deposit", "amount": 1000000}), 201)

	const workers = 6
	statuses := make([]int, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			statuses[i] = call(t, "POST", path, u.Access, map[string]any{"type": "withdraw", "amount": 300000}).Status
		}(i)
	}
	wg.Wait()

	ok := 0
	for _, s := range statuses {
		switch s {
		case 201:
			ok++
		case 422:
		default:
			t.Errorf("status tak terduga: %d", s)
		}
	}
	saved := num(call(t, "GET", "/savings-goals/"+id, u.Access, nil).Body, "goal", "saved_amount")
	if saved < 0 || ok > 3 || int64(ok)*300000+saved != 1000000 {
		t.Fatalf("saldo tidak konsisten: berhasil=%d saldo=%d (harus berhasil<=3 dan berhasil*300000+saldo=1000000)", ok, saved)
	}
}

// ---------- Analitik ----------

func TestAnalytics(t *testing.T) {
	u := newUser(t)
	salary := categoryID(t, u.Access, "income", "Gaji")
	mkTx(t, u.Access, "income", 1000000, salary)

	s := call(t, "GET", "/analytics/summary", u.Access, nil)
	wantStatus(t, s, 200)
	if num(s.Body, "total_income") != 1000000 || num(s.Body, "net") != 1000000 {
		t.Errorf("ringkasan salah: %v", s.Body)
	}

	cmp := call(t, "GET", "/analytics/compare", u.Access, nil)
	wantStatus(t, cmp, 200)
	if num(cmp.Body, "change", "income", "amount") != 1000000 {
		t.Error("selisih pemasukan harus 1.000.000")
	}
	if dig(cmp.Body, "change", "income", "percent") != nil {
		t.Error("persentase harus null jika periode sebelumnya 0")
	}

	wantStatus(t, call(t, "GET", "/analytics/trend?granularity=day", u.Access, nil), 200)
	wantStatus(t, call(t, "GET", "/analytics/trend?granularity=month", u.Access, nil), 200)
	wantStatus(t, call(t, "GET", "/analytics/summary?range=custom&start_date=2026-01-01&end_date=2026-03-31", u.Access, nil), 200)

	for _, path := range []string{
		"/analytics/summary?range=foo",
		"/analytics/summary?range=custom",
		"/analytics/summary?range=custom&start_date=2026-10-10&end_date=2026-10-01",
		"/analytics/summary?range=custom&start_date=2026-01-01&end_date=2027-12-31",
		"/analytics/trend?granularity=tahun",
	} {
		if r := call(t, "GET", path, u.Access, nil); r.Status != 400 {
			t.Errorf("GET %s: status = %d, want 400", path, r.Status)
		}
	}
}

// ---------- Hapus akun ----------

func TestDeleteAccount(t *testing.T) {
	u := newUser(t)
	food := categoryID(t, u.Access, "expense", "Makanan dan minuman")
	mkTx(t, u.Access, "expense", 10000, food)
	call(t, "POST", "/budgets", u.Access, map[string]any{"category_id": food, "period": thisMonth(), "limit_amount": 50000})
	goal := mkGoal(t, u.Access, 100000)
	wantStatus(t, call(t, "POST", "/savings-goals/"+goal+"/contributions", u.Access, map[string]any{"type": "deposit", "amount": 5000}), 201)

	wantStatus(t, call(t, "DELETE", "/auth/me", u.Access, map[string]any{"password": "salah12345"}), 401)
	wantStatus(t, call(t, "GET", "/transactions", u.Access, nil), 200) // data masih ada

	wantStatus(t, call(t, "DELETE", "/auth/me", u.Access, map[string]any{"password": u.Password}), 204)

	wantStatus(t, call(t, "POST", "/auth/login", "", map[string]any{"email": u.Email, "password": u.Password}), 401)
	wantStatus(t, call(t, "POST", "/auth/refresh", "", map[string]any{"refresh_token": u.Refresh}), 401)
	wantStatus(t, call(t, "GET", "/auth/me", u.Access, nil), 404)

	oid, _ := bson.ObjectIDFromHex(u.ID)
	ctx := context.Background()
	if n, _ := db.Collection("users").CountDocuments(ctx, bson.M{"_id": oid}); n != 0 {
		t.Error("dokumen user masih ada")
	}
	for _, name := range []string{"transactions", "budgets", "savings_goals", "savings_contributions", "sessions"} {
		if n, _ := db.Collection(name).CountDocuments(ctx, bson.M{"user_id": oid}); n != 0 {
			t.Errorf("koleksi %s masih berisi %d dokumen milik user yang dihapus", name, n)
		}
	}
}
