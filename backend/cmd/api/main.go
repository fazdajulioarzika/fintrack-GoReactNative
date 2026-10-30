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
	"fintrack-backend/internal/server"
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

	if cfg.SkipDBSetup {
		log.Println("SKIP_DB_SETUP aktif: pembuatan indeks dilewati")
	} else {
		if err := database.Setup(ctx, db); err != nil {
			log.Fatalf("setup database gagal: %v", err)
		}
		log.Println("MongoDB terhubung, indeks siap")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.NewRouter(cfg, client, db),
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
