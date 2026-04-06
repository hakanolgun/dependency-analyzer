package engine

import "math"

var DRSEWeights = Weights{
	Native:          0.40,
	Volume:          0.10,
	APISurface:      0.10,
	Entanglement:    0.15,
	LogicComplexity: 0.25,
}

func Clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func ScoreLabel(normalized float64) string {
	switch {
	case normalized <= 0.30:
		return "EASY"
	case normalized <= 0.70:
		return "MEDIUM"
	default:
		return "HARD"
	}
}

func VolumeScore(sloc int) float64 {
	switch {
	case sloc < 1000:
		return 0.0
	case sloc < 2000:
		return 0.3
	case sloc <= 10000:
		return 0.7
	default:
		return 1.0
	}
}

func ComputeNormalized(m Metrics) float64 {
	total := m.Native*DRSEWeights.Native +
		m.Volume*DRSEWeights.Volume +
		m.APISurface*DRSEWeights.APISurface +
		m.Entanglement*DRSEWeights.Entanglement +
		m.LogicComplexity*DRSEWeights.LogicComplexity

	return Clamp(total)
}

func ToPercentageScore(normalized float64) int {
	return int(math.Round(Clamp(normalized) * 100))
}
