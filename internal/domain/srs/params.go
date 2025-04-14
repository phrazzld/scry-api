package srs

import (
	"github.com/phrazzld/scry-api/internal/domain"
)

// Params defines all configurable parameters for the SRS algorithm
type Params struct {
	// Core limits
	MinEaseFactor float64
	MaxEaseFactor float64

	// Adjustments for different review outcomes
	EaseFactorAdjustment map[domain.ReviewOutcome]float64
	IntervalModifier     map[domain.ReviewOutcome]float64

	// Special case handling
	FirstReviewIntervals map[domain.ReviewOutcome]int
	AgainReviewMinutes   int
}

// ParamsConfig allows overriding the default parameters when creating a new Params instance
type ParamsConfig struct {
	// Core limits
	MinEaseFactor float64
	MaxEaseFactor float64

	// Ease factor adjustments
	AgainEaseFactorAdjustment float64
	HardEaseFactorAdjustment  float64
	GoodEaseFactorAdjustment  float64
	EasyEaseFactorAdjustment  float64

	// Interval modifiers
	AgainIntervalModifier float64
	HardIntervalModifier  float64
	GoodIntervalModifier  float64
	EasyIntervalModifier  float64

	// First review intervals
	FirstReviewHardInterval int
	FirstReviewGoodInterval int
	FirstReviewEasyInterval int

	// Special timing
	AgainReviewMinutes int
}

// NewDefaultParams creates a new Params instance with default values
func NewDefaultParams() *Params {
	return &Params{
		MinEaseFactor: 1.3,
		MaxEaseFactor: 2.5,

		// Default ease factor adjustments
		EaseFactorAdjustment: map[domain.ReviewOutcome]float64{
			domain.ReviewOutcomeAgain: -0.20,
			domain.ReviewOutcomeHard:  -0.15,
			domain.ReviewOutcomeGood:  0.0,
			domain.ReviewOutcomeEasy:  0.15,
		},

		// Default interval modifiers
		IntervalModifier: map[domain.ReviewOutcome]float64{
			domain.ReviewOutcomeAgain: 0.0, // Reset interval
			domain.ReviewOutcomeHard:  1.2, // Slight increase
			domain.ReviewOutcomeGood:  1.0, // Use ease factor directly
			domain.ReviewOutcomeEasy:  1.3, // Significant increase
		},

		// Default first review intervals
		FirstReviewIntervals: map[domain.ReviewOutcome]int{
			domain.ReviewOutcomeHard: 1,
			domain.ReviewOutcomeGood: 1,
			domain.ReviewOutcomeEasy: 2,
		},

		// Review again in 10 minutes
		AgainReviewMinutes: 10,
	}
}

// NewParams creates a new Params instance with custom configuration
func NewParams(config ParamsConfig) *Params {
	params := NewDefaultParams()

	// Override core limits if provided
	if config.MinEaseFactor > 0 {
		params.MinEaseFactor = config.MinEaseFactor
	}
	if config.MaxEaseFactor > 0 {
		params.MaxEaseFactor = config.MaxEaseFactor
	}

	// Override ease factor adjustments if provided
	if config.AgainEaseFactorAdjustment != 0 {
		params.EaseFactorAdjustment[domain.ReviewOutcomeAgain] = config.AgainEaseFactorAdjustment
	}
	if config.HardEaseFactorAdjustment != 0 {
		params.EaseFactorAdjustment[domain.ReviewOutcomeHard] = config.HardEaseFactorAdjustment
	}
	if config.GoodEaseFactorAdjustment != 0 {
		params.EaseFactorAdjustment[domain.ReviewOutcomeGood] = config.GoodEaseFactorAdjustment
	}
	if config.EasyEaseFactorAdjustment != 0 {
		params.EaseFactorAdjustment[domain.ReviewOutcomeEasy] = config.EasyEaseFactorAdjustment
	}

	// Override interval modifiers if provided
	if config.AgainIntervalModifier >= 0 {
		params.IntervalModifier[domain.ReviewOutcomeAgain] = config.AgainIntervalModifier
	}
	if config.HardIntervalModifier > 0 {
		params.IntervalModifier[domain.ReviewOutcomeHard] = config.HardIntervalModifier
	}
	if config.GoodIntervalModifier > 0 {
		params.IntervalModifier[domain.ReviewOutcomeGood] = config.GoodIntervalModifier
	}
	if config.EasyIntervalModifier > 0 {
		params.IntervalModifier[domain.ReviewOutcomeEasy] = config.EasyIntervalModifier
	}

	// Override first review intervals if provided
	if config.FirstReviewHardInterval > 0 {
		params.FirstReviewIntervals[domain.ReviewOutcomeHard] = config.FirstReviewHardInterval
	}
	if config.FirstReviewGoodInterval > 0 {
		params.FirstReviewIntervals[domain.ReviewOutcomeGood] = config.FirstReviewGoodInterval
	}
	if config.FirstReviewEasyInterval > 0 {
		params.FirstReviewIntervals[domain.ReviewOutcomeEasy] = config.FirstReviewEasyInterval
	}

	// Override special timing if provided
	if config.AgainReviewMinutes > 0 {
		params.AgainReviewMinutes = config.AgainReviewMinutes
	}

	return params
}
