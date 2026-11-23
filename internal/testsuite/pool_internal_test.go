package testsuite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPoolReserve(t *testing.T) {
	tests := []struct {
		name      string
		max       int
		growEvery int
		requests  int
		want      []int
	}{
		{
			name:      "success-one-container-until-growth-is-earned",
			max:       3,
			growEvery: 4,
			requests:  3,
			want:      []int{0, 0, 0},
		},
		{
			name:      "success-grows-on-the-request-that-earns-it",
			max:       3,
			growEvery: 2,
			requests:  5,
			want:      []int{0, 1, 0, 0, 1},
		},
		{
			name:      "success-stops-at-the-cap",
			max:       2,
			growEvery: 1,
			requests:  6,
			want:      []int{0, 1, 0, 1, 0, 1},
		},
		{
			name:      "success-a-cap-of-one-never-grows",
			max:       1,
			growEvery: 1,
			requests:  4,
			want:      []int{0, 0, 0, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &pool{
				size:      1,
				max:       tt.max,
				growEvery: tt.growEvery,
			}

			got := make([]int, 0, tt.requests)
			for range tt.requests {
				got = append(got, p.reserve())
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{
			name: "success-unset-uses-the-default",
			raw:  "",
			want: 3,
		},
		{
			name: "success-in-range-is-kept",
			raw:  "5",
			want: 5,
		},
		{
			name: "success-above-the-ceiling-is-clamped",
			raw:  "99",
			want: 10,
		},
		{
			name: "error-not-a-number-falls-back-to-the-default",
			raw:  "three",
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SPINE_TEST_ENVINT", tt.raw)
			require.Equal(t, tt.want, envInt("SPINE_TEST_ENVINT", 3, 1, 10))
		})
	}
}
