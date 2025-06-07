package srs

import (
	"testing"

	"github.com/phrazzld/scry-api/internal/domain"
)

func TestNewDefaultParams(t *testing.T) {
	t.Parallel() // Enable parallel execution
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
	t.Parallel() // Enable parallel execution
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
		AgainReviewMinutes:        20,
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

	if customParams.AgainReviewMinutes != 20 {
		t.Errorf("AgainReviewMinutes not set correctly, got %d, expected 20",
			customParams.AgainReviewMinutes)
	}
}

func TestNewParamsPartialConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config ParamsConfig
		check  func(t *testing.T, params *Params)
	}{
		{
			name: "only_again_review_minutes",
			config: ParamsConfig{
				AgainReviewMinutes: 15,
			},
			check: func(t *testing.T, params *Params) {
				if params.AgainReviewMinutes != 15 {
					t.Errorf("AgainReviewMinutes not set correctly, got %d, expected 15",
						params.AgainReviewMinutes)
				}
				// Check defaults are preserved
				defaultParams := NewDefaultParams()
				if params.MinEaseFactor != defaultParams.MinEaseFactor {
					t.Errorf("MinEaseFactor should use default value")
				}
			},
		},
		{
			name: "zero_values_not_applied",
			config: ParamsConfig{
				MinEaseFactor:      0, // Should not override default
				AgainReviewMinutes: 0, // Should not override default
			},
			check: func(t *testing.T, params *Params) {
				defaultParams := NewDefaultParams()
				if params.MinEaseFactor != defaultParams.MinEaseFactor {
					t.Errorf("MinEaseFactor should use default when config value is 0")
				}
				if params.AgainReviewMinutes != defaultParams.AgainReviewMinutes {
					t.Errorf("AgainReviewMinutes should use default when config value is 0")
				}
			},
		},
		{
			name: "partial_ease_factor_adjustments",
			config: ParamsConfig{
				AgainEaseFactorAdjustment: -0.5,
				// Other ease factor adjustments not set
			},
			check: func(t *testing.T, params *Params) {
				if params.EaseFactorAdjustment[domain.ReviewOutcomeAgain] != -0.5 {
					t.Errorf("Again ease factor adjustment not set correctly")
				}
				// Check other adjustments use defaults
				defaultParams := NewDefaultParams()
				if params.EaseFactorAdjustment[domain.ReviewOutcomeHard] != defaultParams.EaseFactorAdjustment[domain.ReviewOutcomeHard] {
					t.Errorf("Hard ease factor adjustment should use default")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := NewParams(tt.config)
			tt.check(t, params)
		})
	}
}
