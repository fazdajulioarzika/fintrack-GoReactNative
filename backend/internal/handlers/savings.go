package handlers

import (
	"context"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"fintrack-backend/internal/httpx"
	"fintrack-backend/internal/middleware"
	"fintrack-backend/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const maxMoney int64 = 999999999999

var (
	errGoalNotFound = errors.New("goal not found")
	errInsufficient = errors.New("insufficient saved amount")
	errOverLimit    = errors.New("saved amount over limit")
)

type SavingsHandler struct {
	client   *mongo.Client // dibutuhkan untuk memulai transaksi database
	goals    *mongo.Collection
	contribs *mongo.Collection
}

func NewSavingsHandler(client *mongo.Client, db *mongo.Database) *SavingsHandler {
	return &SavingsHandler{
		client:   client,
		goals:    db.Collection("savings_goals"),
		contribs: db.Collection("savings_contributions"),
	}
}

// ---------- Request & response ----------

type createGoalReq struct {
	Name         string     `json:"name" binding:"required,max=100"`
	TargetAmount int64      `json:"target_amount" binding:"required,gt=0,lte=999999999999"`
	TargetDate   *time.Time `json:"target_date"`
	Icon         string     `json:"icon" binding:"max=50"`
}

type updateGoalReq struct {
	Name            *string    `json:"name" binding:"omitempty,max=100"`
	TargetAmount    *int64     `json:"target_amount" binding:"omitempty,gt=0,lte=999999999999"`
	TargetDate      *time.Time `json:"target_date"`
	ClearTargetDate bool       `json:"clear_target_date"`
	Icon            *string    `json:"icon" binding:"omitempty,max=50"`
}

type contributeReq struct {
	Type             string     `json:"type" binding:"required,oneof=deposit withdraw"`
	Amount           int64      `json:"amount" binding:"required,gt=0,lte=999999999999"`
	ContributionDate *time.Time `json:"contribution_date"`
	Note             string     `json:"note" binding:"max=500"`
}

type goalDTO struct {
	ID              bson.ObjectID `json:"id"`
	Name            string        `json:"name"`
	TargetAmount    int64         `json:"target_amount"`
	SavedAmount     int64         `json:"saved_amount"`
	RemainingAmount int64         `json:"remaining_amount"`
	ProgressPercent float64       `json:"progress_percent"` // maksimal 100
	TargetDate      *time.Time    `json:"target_date"`
	Icon            string        `json:"icon"`
	Status          string        `json:"status"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

type contributionDTO struct {
	ID               bson.ObjectID `json:"id"`
	GoalID           bson.ObjectID `json:"goal_id"`
	Type             string        `json:"type"`
	Amount           int64         `json:"amount"`
	ContributionDate time.Time     `json:"contribution_date"`
	Note             string        `json:"note"`
	CreatedAt        time.Time     `json:"created_at"`
}

// goalProgress = terkumpul / target x 100, dibatasi 0-100, satu desimal.
// Memakai aritmetika integer dulu agar tidak ada kesalahan pembulatan float.
func goalProgress(saved, target int64) float64 {
	if target <= 0 {
		return 0
	}
	p := saved * 1000 / target
	if p < 0 {
		p = 0
	}
	if p > 1000 {
		p = 1000
	}
	return float64(p) / 10
}

func toGoalDTO(g models.SavingsGoal) goalDTO {
	remaining := g.TargetAmount - g.SavedAmount
	if remaining < 0 {
		remaining = 0
	}
	return goalDTO{
		ID: g.ID, Name: g.Name,
		TargetAmount: g.TargetAmount, SavedAmount: g.SavedAmount,
		RemainingAmount: remaining,
		ProgressPercent: goalProgress(g.SavedAmount, g.TargetAmount),
		TargetDate:      g.TargetDate, Icon: g.Icon, Status: g.Status,
		CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt,
	}
}

func toContributionDTO(c models.SavingsContribution) contributionDTO {
	return contributionDTO{
		ID: c.ID, GoalID: c.GoalID, Type: c.Type, Amount: c.Amount,
		ContributionDate: c.ContributionDate, Note: c.Note, CreatedAt: c.CreatedAt,
	}
}

// ---------- Handlers: target ----------

func (h *SavingsHandler) List(c *gin.Context) {
	uid := middleware.UserID(c)
	ctx := c.Request.Context()
	filter := bson.M{"user_id": uid}

	if s := c.Query("status"); s != "" {
		if s != models.GoalActive && s != models.GoalAchieved {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "status harus active atau achieved")
			return
		}
		filter["status"] = s
	}

	cur, err := h.goals.Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	var goals []models.SavingsGoal
	if err := cur.All(ctx, &goals); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	out := make([]goalDTO, 0, len(goals))
	var totalTarget, totalSaved int64
	for _, g := range goals {
		out = append(out, toGoalDTO(g))
		totalTarget += g.TargetAmount
		totalSaved += g.SavedAmount
	}
	c.JSON(http.StatusOK, gin.H{"total_target": totalTarget, "total_saved": totalSaved, "data": out})
}

func (h *SavingsHandler) Create(c *gin.Context) {
	var req createGoalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error",
			"Data tidak valid: name wajib diisi, target_amount harus bilangan bulat lebih dari 0")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Nama target tidak boleh kosong")
		return
	}

	now := time.Now().UTC()
	g := models.SavingsGoal{
		ID:           bson.NewObjectID(),
		UserID:       middleware.UserID(c),
		Name:         name,
		TargetAmount: req.TargetAmount,
		Icon:         strings.TrimSpace(req.Icon),
		Status:       models.GoalActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if req.TargetDate != nil && !req.TargetDate.IsZero() {
		d := req.TargetDate.UTC()
		g.TargetDate = &d
	}

	if _, err := h.goals.InsertOne(c.Request.Context(), g); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"goal": toGoalDTO(g)})
}

func (h *SavingsHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	g, err := h.findGoal(c.Request.Context(), middleware.UserID(c), id)
	if errors.Is(err, mongo.ErrNoDocuments) {
		httpx.Error(c, http.StatusNotFound, "not_found", "Target tabungan tidak ditemukan")
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(http.StatusOK, gin.H{"goal": toGoalDTO(*g)})
}

func (h *SavingsHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	var req updateGoalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Data tidak valid")
		return
	}

	set := bson.M{"updated_at": time.Now().UTC()}
	update := bson.M{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			httpx.Error(c, http.StatusBadRequest, "validation_error", "Nama target tidak boleh kosong")
			return
		}
		set["name"] = name
	}
	if req.TargetAmount != nil {
		set["target_amount"] = *req.TargetAmount
	}
	if req.Icon != nil {
		set["icon"] = strings.TrimSpace(*req.Icon)
	}
	if req.ClearTargetDate {
		update["$unset"] = bson.M{"target_date": ""}
	} else if req.TargetDate != nil && !req.TargetDate.IsZero() {
		set["target_date"] = req.TargetDate.UTC()
	}
	update["$set"] = set

	filter := bson.M{"_id": id, "user_id": uid}
	res, err := h.goals.UpdateOne(ctx, filter, update)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	if res.MatchedCount == 0 {
		httpx.Error(c, http.StatusNotFound, "not_found", "Target tabungan tidak ditemukan")
		return
	}

	// Target berubah -> status (active/achieved) harus dihitung ulang
	if req.TargetAmount != nil {
		if err := h.recomputeStatus(ctx, filter); err != nil {
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
			return
		}
	}

	g, err := h.findGoal(ctx, uid, id)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.JSON(http.StatusOK, gin.H{"goal": toGoalDTO(*g)})
}

// Delete menghapus target beserta seluruh riwayat kontribusinya dalam satu transaksi.
func (h *SavingsHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	sess, err := h.client.StartSession()
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	defer sess.EndSession(ctx)

	_, err = sess.WithTransaction(ctx, func(tctx context.Context) (any, error) {
		res, err := h.goals.DeleteOne(tctx, bson.M{"_id": id, "user_id": uid})
		if err != nil {
			return nil, err
		}
		if res.DeletedCount == 0 {
			return nil, errGoalNotFound
		}
		_, err = h.contribs.DeleteMany(tctx, bson.M{"goal_id": id, "user_id": uid})
		return nil, err
	})
	if errors.Is(err, errGoalNotFound) {
		httpx.Error(c, http.StatusNotFound, "not_found", "Target tabungan tidak ditemukan")
		return
	}
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------- Handlers: kontribusi ----------

func (h *SavingsHandler) ListContributions(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	if _, err := h.findGoal(ctx, uid, id); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			httpx.Error(c, http.StatusNotFound, "not_found", "Target tabungan tidak ditemukan")
			return
		}
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}

	page := intQuery(c, "page", 1, 1, 1_000_000)
	limit := intQuery(c, "limit", 20, 1, 100)
	filter := bson.M{"goal_id": id, "user_id": uid}

	total, err := h.contribs.CountDocuments(ctx, filter)
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	cur, err := h.contribs.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "contribution_date", Value: -1}, {Key: "_id", Value: -1}}).
		SetSkip(int64((page-1)*limit)).
		SetLimit(int64(limit)))
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	var list []models.SavingsContribution
	if err := cur.All(ctx, &list); err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	out := make([]contributionDTO, 0, len(list))
	for _, ct := range list {
		out = append(out, toContributionDTO(ct))
	}
	c.JSON(http.StatusOK, gin.H{
		"data": out,
		"meta": gin.H{"page": page, "limit": limit, "total": total},
	})
}

// Contribute menambah/mengurangi saldo target dan mencatat riwayatnya secara atomik.
func (h *SavingsHandler) Contribute(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	uid := middleware.UserID(c)
	ctx := c.Request.Context()

	var req contributeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "validation_error",
			"Data tidak valid: type (deposit/withdraw) dan amount (bilangan bulat > 0) wajib diisi")
		return
	}

	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if len(key) > 100 {
		httpx.Error(c, http.StatusBadRequest, "validation_error", "Idempotency-Key maksimal 100 karakter")
		return
	}
	if key != "" {
		if h.respondExisting(c, uid, key) {
			return
		}
	}

	now := time.Now().UTC()
	date := now
	if req.ContributionDate != nil && !req.ContributionDate.IsZero() {
		date = req.ContributionDate.UTC()
	}
	contrib := models.SavingsContribution{
		ID:               bson.NewObjectID(),
		UserID:           uid,
		GoalID:           id,
		Type:             req.Type,
		Amount:           req.Amount,
		ContributionDate: date,
		Note:             strings.TrimSpace(req.Note),
		IdempotencyKey:   key,
		CreatedAt:        now,
	}

	sess, err := h.client.StartSession()
	if err != nil {
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		return
	}
	defer sess.EndSession(ctx)

	var updated models.SavingsGoal
	_, err = sess.WithTransaction(ctx, func(tctx context.Context) (any, error) {
		filter := bson.M{"_id": id, "user_id": uid}
		delta := req.Amount
		if req.Type == models.ContribWithdraw {
			delta = -req.Amount
			filter["saved_amount"] = bson.M{"$gte": req.Amount} // tidak boleh jadi negatif
		} else {
			filter["saved_amount"] = bson.M{"$lte": maxMoney - req.Amount} // cegah overflow
		}

		// Pipeline update: dua tahap berurutan, tahap kedua sudah melihat saved_amount baru.
		pipeline := mongo.Pipeline{
			bson.D{{Key: "$set", Value: bson.M{
				"saved_amount": bson.M{"$add": bson.A{"$saved_amount", delta}},
			}}},
			bson.D{{Key: "$set", Value: bson.M{
				"status": bson.M{"$cond": bson.A{
					bson.M{"$gte": bson.A{"$saved_amount", "$target_amount"}},
					models.GoalAchieved, models.GoalActive,
				}},
				"updated_at": now,
			}}},
		}
		err := h.goals.FindOneAndUpdate(tctx, filter, pipeline,
			options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&updated)
		if errors.Is(err, mongo.ErrNoDocuments) {
			n, cerr := h.goals.CountDocuments(tctx, bson.M{"_id": id, "user_id": uid})
			if cerr != nil {
				return nil, cerr
			}
			switch {
			case n == 0:
				return nil, errGoalNotFound
			case req.Type == models.ContribWithdraw:
				return nil, errInsufficient
			default:
				return nil, errOverLimit
			}
		}
		if err != nil {
			return nil, err
		}

		_, err = h.contribs.InsertOne(tctx, contrib)
		return nil, err
	})

	switch {
	case errors.Is(err, errGoalNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "Target tabungan tidak ditemukan")
	case errors.Is(err, errInsufficient):
		httpx.Error(c, http.StatusUnprocessableEntity, "insufficient_savings", "Penarikan melebihi jumlah yang terkumpul")
	case errors.Is(err, errOverLimit):
		httpx.Error(c, http.StatusBadRequest, "amount_too_large", "Jumlah tabungan melebihi batas yang diizinkan")
	case err != nil && key != "" && mongo.IsDuplicateKeyError(err):
		// Request kembar masuk bersamaan: transaksi dibatalkan, kembalikan hasil request pertama.
		if !h.respondExisting(c, uid, key) {
			httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
		}
	case err != nil:
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan server")
	default:
		c.JSON(http.StatusCreated, gin.H{
			"goal":         toGoalDTO(updated),
			"contribution": toContributionDTO(contrib),
		})
	}
}

// ---------- Helper ----------

func (h *SavingsHandler) findGoal(ctx context.Context, uid, id bson.ObjectID) (*models.SavingsGoal, error) {
	var g models.SavingsGoal
	if err := h.goals.FindOne(ctx, bson.M{"_id": id, "user_id": uid}).Decode(&g); err != nil {
		return nil, err
	}
	return &g, nil
}

func (h *SavingsHandler) recomputeStatus(ctx context.Context, filter bson.M) error {
	_, err := h.goals.UpdateOne(ctx, filter, mongo.Pipeline{
		bson.D{{Key: "$set", Value: bson.M{
			"status": bson.M{"$cond": bson.A{
				bson.M{"$gte": bson.A{"$saved_amount", "$target_amount"}},
				models.GoalAchieved, models.GoalActive,
			}},
		}}},
	})
	return err
}

// respondExisting: jika kontribusi dengan key ini sudah pernah dibuat, kirim hasilnya (200).
func (h *SavingsHandler) respondExisting(c *gin.Context, uid bson.ObjectID, key string) bool {
	ctx := c.Request.Context()
	var existing models.SavingsContribution
	if err := h.contribs.FindOne(ctx, bson.M{"user_id": uid, "idempotency_key": key}).Decode(&existing); err != nil {
		return false
	}
	g, err := h.findGoal(ctx, uid, existing.GoalID)
	if err != nil {
		httpx.Error(c, http.StatusNotFound, "not_found", "Target tabungan tidak ditemukan")
		return true
	}
	c.JSON(http.StatusOK, gin.H{
		"goal":         toGoalDTO(*g),
		"contribution": toContributionDTO(existing),
	})
	return true
}

var _ = math.Round // (dipakai di fitur lanjutan; hapus jika tidak perlu)
