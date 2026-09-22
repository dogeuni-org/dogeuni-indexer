package utils

import "testing"

func TestPageLimit(t *testing.T) {
	for _, c := range []struct{ in, want int }{{-1, 100}, {0, 0}, {10, 10}, {100, 100}, {101, 100}, {1 << 30, 100}} {
		if got := PageLimit(c.in, 100); got != c.want {
			t.Errorf("PageLimit(%d) = %d, want %d", c.in, got, c.want)
		}
	}
	if got := PageOffset(int64(-5)); got != 0 {
		t.Errorf("PageOffset(-5) = %d", got)
	}
	if got := PageOffset(7); got != 7 {
		t.Errorf("PageOffset(7) = %d", got)
	}
}
