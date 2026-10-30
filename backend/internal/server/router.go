// Package server merakit router HTTP. Dipakai oleh main dan oleh tes integrasi.
package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"fintrack-backend/internal/config"
	"fintrack-backend/internal/handlers"
	"fintrack-backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/time/rate"
)

// logFormat mencatat request TANPA query string (bisa berisi kata kunci pencarian pribadi).
func logFormat(p gin.LogFormatterParams) string {
	path, _, _ := strings.Cut(p.Path, "?")
	return fmt.Sprintf("%s | %3d | %12v | %-15s | %-7s %s\n",
		p.TimeStamp.Format("2006/01/02 15:04:05"), p.StatusCode, p.Latency, p.ClientIP, p.Method, path)
}

func NewRouter(cfg *config.Config, client *mongo.Client, db *mongo.Database) *gin.Engine {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(
		gin.LoggerWithFormatter(logFormat),
		gin.Recovery(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.CORSOrigins),
		middleware.MaxBody(1<<20), // 1 MB
	)

	// Di belakang proxy hosting, isi TRUSTED_PROXIES agar IP asli pengguna terbaca (untuk rate limit).
	if len(cfg.TrustedProxies) > 0 {
		_ = r.SetTrustedProxies(cfg.TrustedProxies)
	} else {
		_ = r.SetTrustedProxies(nil)
	}

	// Liveness: proses hidup. Readiness: database juga terjangkau.
	r.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/ready", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if err := client.Ping(ctx, nil); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	authH := handlers.NewAuthHandler(db, cfg)
	catH := handlers.NewCategoryHandler(db)
	txH := handlers.NewTransactionHandler(db)
	dashH := handlers.NewDashboardHandler(db)
	budH := handlers.NewBudgetHandler(db)
	savH := handlers.NewSavingsHandler(client, db)
	anaH := handlers.NewAnalyticsHandler(db)

	perMin := cfg.AuthRatePerMin
	if perMin <= 0 {
		perMin = 10
	}
	burst := cfg.AuthBurst
	if burst <= 0 {
		burst = 5
	}
	limiter := middleware.NewIPRateLimiter(rate.Every(time.Minute/time.Duration(perMin)), burst)

	v1 := r.Group("/api/v1")

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
	api.DELETE("/auth/me", limiter.Middleware(), authH.DeleteMe) // memeriksa password: batasi percobaan

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

	return r
}
