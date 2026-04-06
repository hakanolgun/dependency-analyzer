package engine

import "testing"

func TestVolumeScoreBuckets(t *testing.T) {
	cases := []struct {
		in   int
		want float64
	}{
		{in: 100, want: 0.0},
		{in: 1500, want: 0.3},
		{in: 5000, want: 0.7},
		{in: 15000, want: 1.0},
	}

	for _, tc := range cases {
		got := VolumeScore(tc.in)
		if got != tc.want {
			t.Fatalf("VolumeScore(%d)=%v want=%v", tc.in, got, tc.want)
		}
	}
}

func TestComputeNormalizedRange(t *testing.T) {
	m := Metrics{
		Native:          1,
		Volume:          1,
		APISurface:      1,
		Entanglement:    1,
		LogicComplexity: 1,
	}
	got := ComputeNormalized(m)
	if got < 0 || got > 1 {
		t.Fatalf("normalized out of range: %v", got)
	}
	if got != 1 {
		t.Fatalf("expected max normalized to be 1, got %v", got)
	}
}
