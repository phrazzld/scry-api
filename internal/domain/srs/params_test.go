package srs

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/domain"
)

func TestNewDefaultParams(t *testing.T) {
	params := NewDefaultParams()

	// Verify minimum parameters are set
	if params.MinEaseFactor <= 0 {
		t.Errorf("MinEaseFactor should be positive, got %f", params.MinEaseFactor)
	}

	if params.MaxEaseFactor <= params.MinEaseFactor {
		t.Errorf("MaxEaseFactor should be greater than MinEaseFactor, got %f and %f",
			params.MaxEaseFactor, params.MinEaseFactor)
	}

	// Check outcome-specific parameters exist for all outcomes
	outcomes := []domain.ReviewOutcome{
		domain.ReviewOutcomeAgain,
		domain.ReviewOutcomeHard,
		domain.ReviewOutcomeGood,
		domain.ReviewOutcomeEasy,
	}

	for _, outcome := range outcomes {
		if _, exists := params.EaseFactorAdjustment[outcome]; !exists {
			t.Errorf("EaseFactorAdjustment missing for outcome %s", outcome)
		}

		if _, exists := params.IntervalModifier[outcome]; !exists {
			t.Errorf("IntervalModifier missing for outcome %s", outcome)
		}
	}

	// Check reasonable values
	if params.EaseFactorAdjustment[domain.ReviewOutcomeAgain] >= 0 {
		t.Errorf("EaseFactorAdjustment for Again should be negative, got %f",
			params.EaseFactorAdjustment[domain.ReviewOutcomeAgain])
	}

	if params.EaseFactorAdjustment[domain.ReviewOutcomeEasy] <= 0 {
		t.Errorf("EaseFactorAdjustment for Easy should be positive, got %f",
			params.EaseFactorAdjustment[domain.ReviewOutcomeEasy])
	}

	if params.IntervalModifier[domain.ReviewOutcomeAgain] != 0 {
		t.Errorf("IntervalModifier for Again should be 0, got %f",
			params.IntervalModifier[domain.ReviewOutcomeAgain])
	}

	if params.IntervalModifier[domain.ReviewOutcomeEasy] <= 1.0 {
		t.Errorf("IntervalModifier for Easy should be greater than 1.0, got %f",
			params.IntervalModifier[domain.ReviewOutcomeEasy])
	}
}

func TestNewParams(t *testing.T) {
	customParams := NewParams(ParamsConfig{
		MinEaseFactor:             1.5,
		MaxEaseFactor:             3.0,
		AgainEaseFactorAdjustment: -0.3,
		HardEaseFactorAdjustment:  -0.2,
		GoodEaseFactorAdjustment:  0.0,
		EasyEaseFactorAdjustment:  0.2,
		AgainIntervalModifier:     0.0,
		HardIntervalModifier:      1.1,
		GoodIntervalModifier:      1.7,
		EasyIntervalModifier:      2.2,
		FirstReviewHardInterval:   2,
		FirstReviewGoodInterval:   3,
		FirstReviewEasyInterval:   4,
	})

	// Check custom values were applied
	if customParams.MinEaseFactor != 1.5 {
		t.Errorf(
			"MinEaseFactor not set correctly, got %f, expected 1.5",
			customParams.MinEaseFactor,
		)
	}

	if customParams.MaxEaseFactor != 3.0 {
		t.Errorf(
			"MaxEaseFactor not set correctly, got %f, expected 3.0",
			customParams.MaxEaseFactor,
		)
	}

	if customParams.EaseFactorAdjustment[domain.ReviewOutcomeAgain] != -0.3 {
		t.Errorf("Again ease factor adjustment not set correctly, got %f, expected -0.3",
			customParams.EaseFactorAdjustment[domain.ReviewOutcomeAgain])
	}

	if customParams.IntervalModifier[domain.ReviewOutcomeEasy] != 2.2 {
		t.Errorf("Easy interval modifier not set correctly, got %f, expected 2.2",
			customParams.IntervalModifier[domain.ReviewOutcomeEasy])
	}

	if customParams.FirstReviewIntervals[domain.ReviewOutcomeHard] != 2 {
		t.Errorf("First review Hard interval not set correctly, got %d, expected 2",
			customParams.FirstReviewIntervals[domain.ReviewOutcomeHard])
	}
}
