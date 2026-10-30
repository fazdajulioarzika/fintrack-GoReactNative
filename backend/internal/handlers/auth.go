// Package handlers berisi fungsi yang menangani request HTTP.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"fintrack-backend/internal/auth"
	"fintrack-backend/internal/config"
	"fintrack-backend/internal/httpx"
	"fintrack-backend/internal/middleware"
	"fintrack-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

type AuthHandler struct {
	db        *mongo.Database
	users     *mongo.Collection
	sessions  *mongo.Collection
	cfg       *config.Config
	dummyHash []byte
}

func NewAuthHandler(db *mongo.Database, cfg *config.Config) *AuthHandler {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password"), bcryptCost)
	return &AuthHandler{
		db:        db,
		users:     db.Collection("users"),
		sessions:  db.Collection("sessions"),
		cfg:       cfg,
		dummyHash: dummy,
	}
}

// ---------- Request body ----------
// Tag `binding` dipakai Gin untuk validasi otomatis.

type registerReq struct {
	Name     string `json:"name" binding:"required,min=2,max=100"`
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,min=8,max=72"` // bcrypt hanya membaca 72 byte
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type updateMeReq struct {
	Name     *string `json:"name" binding:"omitempty,min=2,max=100"`
	Timezone *string `json:"timezone"`
}

type deleteMeReq struct {
	Password string `json:"password" binding:"required"`
}

// ---------- Handlers ----------

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error",
			"Data tidak valid: nama minimal 2 karakter, email harus valid, password 8-72 karakter")
		return
	}
	// bcrypt hanya membaca 72 BYTE pertama; karakter non-ASCII (mis. emoji) memakai lebih dari 1 byte.
	if len(req.Password) > 72 {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Password terlalu panjang (maksimal 72 byte)")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	now := time.Now().UTC()
	user := models.User{
		ID:           bson.NewObjectID(),
		Name:         strings.TrimSpace(req.Name),
		Email:        normalizeEmail(req.Email),
		PasswordHash: string(hash),
		Currency:     "IDR",
		Timezone:     "Asia/Jakarta",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if _, err := h.users.InsertOne(c.Request.Context(), user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			httpx.Error(c, http.StatusConflict, "email_taken", "Email sudah terdaftar")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	resp, err := h.issueTokens(c.Request.Context(), &user, c.Request.UserAgent())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Email dan password wajib diisi")
		return
	}

	var user models.User
	err := h.users.FindOne(c.Request.Context(), bson.M{"email": normalizeEmail(req.Email)}).Decode(&user)
	if errors.Is(err, mongo.ErrNoDocuments) {
		_ = bcrypt.CompareHashAndPassword(h.dummyHash, []byte(req.Password)) // samakan waktu respons
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "Email atau password salah")
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "Email atau password salah")
		return
	}

	resp, err := h.issueTokens(c.Request.Context(), &user, c.Request.UserAgent())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Refresh menukar refresh token dengan pasangan token baru (rotasi).
// Token lama langsung dihapus, jadi token yang dicuri dan dipakai ulang akan ditolak.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "refresh_token wajib diisi")
		return
	}

	ctx := c.Request.Context()
	var sess models.Session
	// FindOneAndDelete bersifat atomik: hanya satu request yang bisa memakai token ini.
	err := h.sessions.FindOneAndDelete(ctx,
		bson.M{"token_hash": auth.HashRefreshToken(req.RefreshToken)}).Decode(&sess)
	if err != nil || time.Now().After(sess.ExpiresAt) {
		httpx.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "Sesi tidak valid, silakan login ulang")
		return
	}

	var user models.User
	if err := h.users.FindOne(ctx, bson.M{"_id": sess.UserID}).Decode(&user); err != nil {
		httpx.Error(c, http.StatusUnauthorized, "invalid_refresh_token", "Sesi tidak valid, silakan login ulang")
		return
	}

	resp, err := h.issueTokens(ctx, &user, c.Request.UserAgent())
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Logout mencabut refresh token. Selalu 204 agar tidak membocorkan apakah token valid.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "refresh_token wajib diisi")
		return
	}
	_, _ = h.sessions.DeleteOne(c.Request.Context(),
		bson.M{"token_hash": auth.HashRefreshToken(req.RefreshToken)})
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := h.findCurrentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	var req updateMeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Nama minimal 2 karakter")
		return
	}

	set := bson.M{"updated_at": time.Now().UTC()}
	if req.Name != nil {
		set["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Timezone != nil {
		if _, err := time.LoadLocation(*req.Timezone); err != nil {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "Zona waktu tidak valid (contoh: Asia/Jakarta)")
			return
		}
		set["timezone"] = *req.Timezone
	}

	var user models.User
	err := h.users.FindOneAndUpdate(c.Request.Context(),
		bson.M{"_id": middleware.UserID(c)},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&user)
	if err != nil {
		httpx.Error(c, http.StatusNotFound, "not_found", "Pengguna tidak ditemukan")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

// DeleteMe menghapus akun setelah konfirmasi password.
// Data milik user dihapus lebih dulu. Jika ada yang gagal, akun tetap ada sehingga bisa dicoba lagi.
func (h *AuthHandler) DeleteMe(c *gin.Context) {
	var req deleteMeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Password wajib diisi untuk konfirmasi")
		return
	}
	user, ok := h.findCurrentUser(c)
	if !ok {
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "Password salah")
		return
	}

	ctx := c.Request.Context()
	for _, name := range []string{"transactions", "budgets", "savings_contributions", "savings_goals", "sessions"} {
		if _, err := h.db.Collection(name).DeleteMany(ctx, bson.M{"user_id": user.ID}); err != nil {
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
			return
		}
	}
	if _, err := h.users.DeleteOne(ctx, bson.M{"_id": user.ID}); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- Helper ----------

func (h *AuthHandler) issueTokens(ctx context.Context, user *models.User, userAgent string) (gin.H, error) {
	access, err := auth.NewAccessToken(user.ID.Hex(), h.cfg.JWTSecret, h.cfg.AccessTTL)
	if err != nil {
		return nil, err
	}
	raw, hash, err := auth.NewRefreshToken()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sess := models.Session{
		ID:        bson.NewObjectID(),
		UserID:    user.ID,
		TokenHash: hash,
		UserAgent: userAgent,
		ExpiresAt: now.Add(h.cfg.RefreshTTL),
		CreatedAt: now,
	}
	if _, err := h.sessions.InsertOne(ctx, sess); err != nil {
		return nil, err
	}

	return gin.H{
		"user":          user,
		"access_token":  access,
		"refresh_token": raw,
		"expires_in":    int(h.cfg.AccessTTL.Seconds()),
	}, nil
}

func (h *AuthHandler) findCurrentUser(c *gin.Context) (*models.User, bool) {
	var user models.User
	err := h.users.FindOne(c.Request.Context(), bson.M{"_id": middleware.UserID(c)}).Decode(&user)
	if err != nil {
		httpx.Error(c, http.StatusNotFound, "not_found", "Pengguna tidak ditemukan")
		return nil, false
	}
	return &user, true
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
