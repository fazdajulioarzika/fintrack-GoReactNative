package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata" // menyertakan database zona waktu agar time.LoadLocation selalu bekerja

	"fintrack-backend/internal/config"
	"fintrack-backend/internal/database"
	"fintrack-backend/internal/handlers"
	"fintrack-backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("konfigurasi: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, db, err := database.Connect(ctx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("koneksi MongoDB gagal: %v", err)
	}
	defer client.Disconnect(context.Background())

	if err := database.EnsureIndexes(ctx, db); err != nil {
		log.Fatalf("gagal membuat indeks: %v", err)
	}
	if err := database.EnsureFinanceIndexes(ctx, db); err != nil {
		log.Fatalf("gagal membuat indeks keuangan: %v", err)
	}
	if err := database.SeedCategories(ctx, db); err != nil {
		log.Fatalf("gagal seed kategori: %v", err)
	}
	if err := database.EnsureBudgetIndexes(ctx, db); err != nil {
		log.Fatalf("gagal membuat indeks anggaran: %v", err)
	}
	if err := database.EnsureSavingsIndexes(ctx, db); err != nil {
		log.Fatalf("gagal membuat indeks tabungan: %v", err)
	}
	log.Println("MongoDB terhubung, indeks siap")

	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery()) // Logger Gin tidak mencatat body, jadi password aman
	_ = r.SetTrustedProxies(nil)        // jangan percaya header X-Forwarded-For dari sembarang pihak

	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	authH := handlers.NewAuthHandler(db, cfg)
	catH := handlers.NewCategoryHandler(db)
	txH := handlers.NewTransactionHandler(db)
	dashH := handlers.NewDashboardHandler(db)
	budH := handlers.NewBudgetHandler(db)
	savH := handlers.NewSavingsHandler(client, db)
	anaH := handlers.NewAnalyticsHandler(db)

	// 10 request/menit per IP, burst 5, untuk endpoint yang rawan brute force
	limiter := middleware.NewIPRateLimiter(rate.Every(6*time.Second), 5)

	v1 := r.Group("/api/v1")
	{
		pub := v1.Group("/auth")
		pub.Use(limiter.Middleware())
		pub.POST("/register", authH.Register)
		pub.POST("/login", authH.Login)
		pub.POST("/refresh", authH.Refresh)
		pub.POST("/logout", authH.Logout)

		api := v1.Group("")
		api.Use(middleware.RequireAuth(cfg.JWTSecret))

		api.GET("/auth/me", authH.Me)
		api.PATCH("/auth/me", authH.UpdateMe)
		api.DELETE("/auth/me", authH.DeleteMe)

		api.GET("/categories", catH.List)

		api.GET("/transactions", txH.List)
		api.POST("/transactions", txH.Create)
		api.GET("/transactions/:id", txH.Get)
		api.PATCH("/transactions/:id", txH.Update)
		api.DELETE("/transactions/:id", txH.Delete)

		api.GET("/dashboard/summary", dashH.Summary)
		api.GET("/dashboard/cashflow", dashH.Cashflow)
		api.GET("/dashboard/categories", dashH.Categories)

		api.GET("/budgets", budH.List)
		api.POST("/budgets", budH.Create)
		api.PATCH("/budgets/:id", budH.Update)
		api.DELETE("/budgets/:id", budH.Delete)

		api.GET("/savings-goals", savH.List)
		api.POST("/savings-goals", savH.Create)
		api.GET("/savings-goals/:id", savH.Get)
		api.PATCH("/savings-goals/:id", savH.Update)
		api.DELETE("/savings-goals/:id", savH.Delete)
		api.GET("/savings-goals/:id/contributions", savH.ListContributions)
		api.POST("/savings-goals/:id/contributions", savH.Contribute)

		api.GET("/analytics/summary", anaH.Summary)
		api.GET("/analytics/trend", anaH.Trend)
		api.GET("/analytics/compare", anaH.Compare)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	go func() {
		log.Printf("server berjalan di http://localhost:%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done() // tunggu Ctrl+C
	log.Println("mematikan server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
