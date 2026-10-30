// Perintah ini membuat akun demo berisi data keuangan FIKTIF untuk screenshot dan demo portofolio.
// Jalankan:  go run ./cmd/seed
// Target database mengikuti MONGO_URI dan MONGO_DB di .env. Hanya data milik akun demo yang dihapus/dibuat ulang.
package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"os"
	"time"

	_ "time/tzdata"

	"fintrack-backend/internal/database"
	"fintrack-backend/internal/models"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

const demoEmail = "demo@fintrack.app"

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	_ = godotenv.Load()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		log.Fatal("MONGO_URI wajib diisi (lihat .env)")
	}
	dbName := getenv("MONGO_DB", "fintrack")
	password := getenv("DEMO_PASSWORD", "Demo12345")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, db, err := database.Connect(ctx, uri, dbName)
	if err != nil {
		log.Fatalf("koneksi MongoDB gagal: %v", err)
	}
	defer client.Disconnect(context.Background())
	if err := database.Setup(ctx, db); err != nil {
		log.Fatalf("setup database gagal: %v", err)
	}
	log.Printf("target database: %s", dbName)

	if err := resetDemo(ctx, db); err != nil {
		log.Fatalf("gagal membersihkan data demo lama: %v", err)
	}

	cats, err := loadCategories(ctx, db)
	if err != nil {
		log.Fatalf("gagal memuat kategori: %v", err)
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Fatal(err)
	}
	now := time.Now().In(loc)
	nowUTC := now.UTC()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatal(err)
	}
	user := models.User{
		ID: bson.NewObjectID(), Name: "Demo FinTrack", Email: demoEmail, PasswordHash: string(hash),
		Currency: "IDR", Timezone: "Asia/Jakarta", CreatedAt: nowUTC, UpdatedAt: nowUTC,
	}
	if _, err := db.Collection("users").InsertOne(ctx, user); err != nil {
		log.Fatalf("gagal membuat user: %v", err)
	}

	// ---------- Transaksi ----------
	rnd := rand.New(rand.NewSource(42)) // acak tetap: hasil selalu sama
	var txs []models.Transaction

	add := func(typ, cat string, amount int64, at time.Time, note, pay string) {
		if at.After(now) {
			return // jangan buat transaksi di masa depan
		}
		id, ok := cats[typ+"|"+cat]
		if !ok {
			log.Fatalf("kategori %q (%s) tidak ditemukan", cat, typ)
		}
		txs = append(txs, models.Transaction{
			ID: bson.NewObjectID(), UserID: user.ID, Type: typ, Amount: amount, CategoryID: id,
			Note: note, PaymentMethod: pay, TransactionDate: at.UTC(), CreatedAt: nowUTC, UpdatedAt: nowUTC,
		})
	}
	between := func(lo, hi int64) int64 { return (lo + rnd.Int63n(hi-lo+1)) / 500 * 500 }
	pick := func(xs []string) string { return xs[rnd.Intn(len(xs))] }

	foodNotes := []string{"Makan siang", "Kopi", "Makan malam", "Sarapan", "Nasi goreng", "Bakso", "Es teh", "Ayam geprek"}
	rideNotes := []string{"Gojek", "Bensin", "Parkir", "KRL", "Grab"}
	funNotes := []string{"Nonton bioskop", "Langganan streaming", "Game", "Karaoke"}
	shopNotes := []string{"Alat tulis", "Baju", "Peralatan rumah", "Sabun dan sampo"}
	cash := []string{"Tunai", "E-wallet"}

	for i := 2; i >= 0; i-- {
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -i, 0)
		days := first.AddDate(0, 1, -1).Day()
		at := func(day, hour int) time.Time {
			if day > days {
				day = days
			}
			return time.Date(first.Year(), first.Month(), day, hour, rnd.Intn(60), 0, 0, loc)
		}

		add(models.TypeIncome, "Gaji", 8_500_000, at(25, 9), "Gaji bulanan", "Transfer")
		add(models.TypeIncome, "Freelance", between(1_500_000, 2_500_000), at(12, 15), "Proyek desain", "Transfer")

		add(models.TypeExpense, "Tagihan", 450_000, at(5, 10), "Listrik PLN", "E-wallet")
		add(models.TypeExpense, "Tagihan", 350_000, at(8, 10), "Internet bulanan", "Transfer")
		add(models.TypeExpense, "Pendidikan", 200_000, at(10, 19), "Kursus online", "Kartu")
		add(models.TypeExpense, "Kesehatan", between(100_000, 300_000), at(17, 17), "Periksa dokter", "Tunai")
		add(models.TypeExpense, "Belanja", between(600_000, 900_000), at(2, 16), "Belanja bulanan", "Kartu")

		for d := 1; d <= days; d++ {
			if rnd.Float64() < 0.85 {
				add(models.TypeExpense, "Makanan dan minuman", between(15_000, 70_000), at(d, 8+rnd.Intn(12)), pick(foodNotes), pick(cash))
			}
			if rnd.Float64() < 0.5 {
				add(models.TypeExpense, "Transportasi", between(10_000, 40_000), at(d, 7+rnd.Intn(12)), pick(rideNotes), pick(cash))
			}
			if rnd.Float64() < 0.10 {
				add(models.TypeExpense, "Hiburan", between(30_000, 150_000), at(d, 18+rnd.Intn(4)), pick(funNotes), "E-wallet")
			}
			if rnd.Float64() < 0.10 {
				add(models.TypeExpense, "Belanja", between(50_000, 250_000), at(d, 13+rnd.Intn(6)), pick(shopNotes), "Kartu")
			}
		}
	}
	if len(txs) == 0 {
		log.Fatal("tidak ada transaksi yang dibuat")
	}
	if _, err := db.Collection("transactions").InsertMany(ctx, txs); err != nil {
		log.Fatalf("gagal menyimpan transaksi: %v", err)
	}

	// ---------- Anggaran (bulan berjalan) ----------
	period := now.Format("2006-01")
	var budgets []models.Budget
	for _, b := range []struct {
		cat   string
		limit int64
	}{
		{"Makanan dan minuman", 1_200_000},
		{"Transportasi", 600_000},
		{"Hiburan", 300_000},
		{"Belanja", 1_500_000},
	} {
		budgets = append(budgets, models.Budget{
			ID: bson.NewObjectID(), UserID: user.ID, CategoryID: cats[models.TypeExpense+"|"+b.cat],
			Period: period, LimitAmount: b.limit, CreatedAt: nowUTC, UpdatedAt: nowUTC,
		})
	}
	if _, err := db.Collection("budgets").InsertMany(ctx, budgets); err != nil {
		log.Fatalf("gagal menyimpan anggaran: %v", err)
	}

	// ---------- Target tabungan + riwayat ----------
	type contribSeed struct {
		monthsAgo, day int
		typ            string
		amount         int64
		note           string
	}
	goalSeeds := []struct {
		name        string
		target      int64
		icon        string
		dueInMonths int
		contribs    []contribSeed
	}{
		{"Dana darurat", 15_000_000, "shield-checkmark", 10, []contribSeed{
			{2, 26, models.ContribDeposit, 1_500_000, "Sisa gaji"},
			{1, 26, models.ContribDeposit, 1_500_000, "Sisa gaji"},
			{0, 3, models.ContribDeposit, 1_000_000, "Bonus kecil"},
		}},
		{"Liburan Bali", 6_000_000, "airplane", 6, []contribSeed{
			{2, 27, models.ContribDeposit, 800_000, "Tabungan liburan"},
			{1, 27, models.ContribDeposit, 800_000, "Tabungan liburan"},
			{1, 28, models.ContribWithdraw, 200_000, "Dipakai servis motor"},
			{0, 2, models.ContribDeposit, 500_000, "Tabungan liburan"},
		}},
		{"HP baru", 3_000_000, "phone-portrait", 0, []contribSeed{
			{2, 26, models.ContribDeposit, 1_000_000, "Cicil tabungan"},
			{1, 26, models.ContribDeposit, 1_000_000, "Cicil tabungan"},
			{0, 4, models.ContribDeposit, 1_000_000, "Cicil tabungan"},
		}},
	}

	var goals []models.SavingsGoal
	var contribs []models.SavingsContribution
	for _, gs := range goalSeeds {
		g := models.SavingsGoal{
			ID: bson.NewObjectID(), UserID: user.ID, Name: gs.name, TargetAmount: gs.target, Icon: gs.icon,
			Status: models.GoalActive, CreatedAt: nowUTC, UpdatedAt: nowUTC,
		}
		if gs.dueInMonths > 0 {
			due := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, gs.dueInMonths, 0).UTC()
			g.TargetDate = &due
		}
		for _, c := range gs.contribs {
			first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc).AddDate(0, -c.monthsAgo, 0)
			at := time.Date(first.Year(), first.Month(), c.day, 9, 0, 0, 0, loc)
			if at.After(now) {
				continue
			}
			if c.typ == models.ContribDeposit {
				g.SavedAmount += c.amount
			} else {
				g.SavedAmount -= c.amount
			}
			contribs = append(contribs, models.SavingsContribution{
				ID: bson.NewObjectID(), UserID: user.ID, GoalID: g.ID, Type: c.typ, Amount: c.amount,
				ContributionDate: at.UTC(), Note: c.note, CreatedAt: nowUTC,
			})
		}
		if g.SavedAmount >= g.TargetAmount {
			g.Status = models.GoalAchieved
		}
		goals = append(goals, g)
	}
	if _, err := db.Collection("savings_goals").InsertMany(ctx, goals); err != nil {
		log.Fatalf("gagal menyimpan target tabungan: %v", err)
	}
	if len(contribs) > 0 {
		if _, err := db.Collection("savings_contributions").InsertMany(ctx, contribs); err != nil {
			log.Fatalf("gagal menyimpan kontribusi: %v", err)
		}
	}

	log.Println("selesai.")
	log.Printf("akun demo : %s", demoEmail)
	log.Printf("password  : %s", password)
	log.Printf("data      : %d transaksi, %d anggaran, %d target, %d kontribusi", len(txs), len(budgets), len(goals), len(contribs))
}

func resetDemo(ctx context.Context, db *mongo.Database) error {
	var old models.User
	err := db.Collection("users").FindOne(ctx, bson.M{"email": demoEmail}).Decode(&old)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, name := range []string{"transactions", "budgets", "savings_contributions", "savings_goals", "sessions"} {
		if _, err := db.Collection(name).DeleteMany(ctx, bson.M{"user_id": old.ID}); err != nil {
			return err
		}
	}
	_, err = db.Collection("users").DeleteOne(ctx, bson.M{"_id": old.ID})
	return err
}

// Kunci peta: "jenis|nama kategori", karena "Lainnya" ada di pengeluaran maupun pemasukan.
func loadCategories(ctx context.Context, db *mongo.Database) (map[string]bson.ObjectID, error) {
	cur, err := db.Collection("categories").Find(ctx, bson.M{"user_id": nil})
	if err != nil {
		return nil, err
	}
	var list []models.Category
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	m := make(map[string]bson.ObjectID, len(list))
	for _, c := range list {
		m[c.Type+"|"+c.Name] = c.ID
	}
	return m, nil
}
