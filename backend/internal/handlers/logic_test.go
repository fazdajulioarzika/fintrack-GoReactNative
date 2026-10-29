package handlers

import (
	"testing"
	"time"

	_ "time/tzdata"
)

func TestBudgetStatus(t *testing.T) {
	tests := []struct {
		name         string
		spent, limit int64
		want         string
	}{
		{"nol terpakai", 0, 100000, "normal"},
		{"79 persen", 79999, 100000, "normal"},
		{"tepat 80 persen", 80000, 100000, "warning"},
		{"99 persen", 99999, 100000, "warning"},
		{"tepat 100 persen", 100000, 100000, "exceeded"},
		{"lebih dari batas", 150000, 100000, "exceeded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := budgetStatus(tt.spent, tt.limit); got != tt.want {
				t.Errorf("budgetStatus(%d,%d) = %s, want %s", tt.spent, tt.limit, got, tt.want)
			}
		})
	}
}

func TestGoalProgress(t *testing.T) {
	tests := []struct {
		saved, target int64
		want          float64
	}{
		{0, 1000000, 0},
		{250000, 1000000, 25},
		{333333, 1000000, 33.3},
		{1000000, 1000000, 100},
		{2500000, 1000000, 100}, // tidak boleh melebihi 100
		{-5, 1000000, 0},
		{100, 0, 0}, // target tidak valid tidak boleh menyebabkan pembagian nol
	}
	for _, tt := range tests {
		if got := goalProgress(tt.saved, tt.target); got != tt.want {
			t.Errorf("goalProgress(%d,%d) = %v, want %v", tt.saved, tt.target, got, tt.want)
		}
	}
}

func TestChangePercent(t *testing.T) {
	if got := changePercent(100, 0); got != nil {
		t.Errorf("pembanding 0 harus nil, dapat %v", *got)
	}
	if got := changePercent(150, 100); got == nil || *got != 50 {
		t.Errorf("150 vs 100 harus +50, dapat %v", got)
	}
	if got := changePercent(50, 100); got == nil || *got != -50 {
		t.Errorf("50 vs 100 harus -50, dapat %v", got)
	}
	// Selisih negatif ke nilai yang lebih baik harus bernilai positif
	if got := changePercent(-50, -100); got == nil || *got != 50 {
		t.Errorf("-50 vs -100 harus +50, dapat %v", got)
	}
}

func TestParseMonthBoundariesUseUserTimezone(t *testing.T) {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	p, err := parseMonth("2026-10", jakarta)
	if err != nil {
		t.Fatal(err)
	}
	if p.Label != "2026-10" {
		t.Errorf("label = %s", p.Label)
	}

	inOct := time.Date(2026, 10, 31, 23, 30, 0, 0, jakarta) // 31 Okt malam WIB (masih UTC 16:30)
	inNov := time.Date(2026, 11, 1, 0, 0, 0, 0, jakarta)
	if inOct.Before(p.Start) || !inOct.Before(p.End) {
		t.Error("31 Okt 23:30 WIB harus masuk Oktober")
	}
	if inNov.Before(p.End) {
		t.Error("1 Nov 00:00 WIB tidak boleh masuk Oktober")
	}

	if _, err := parseMonth("2026-13", jakarta); err == nil {
		t.Error("bulan 13 harus ditolak")
	}
	if _, err := parseMonth("Oktober", jakarta); err == nil {
		t.Error("format salah harus ditolak")
	}
}
