package main

import "testing"

func TestScoreSelection(t *testing.T) {
	cases := []struct {
		dice  []int
		score int
		ok    bool
	}{
		{[]int{1}, 100, true},
		{[]int{5}, 50, true},
		{[]int{1, 5}, 150, true},
		{[]int{1, 1}, 200, true},
		{[]int{5, 5}, 100, true},

		{[]int{1, 1, 1}, 1000, true},
		{[]int{1, 1, 1, 1}, 2000, true},
		{[]int{1, 1, 1, 1, 1}, 4000, true},
		{[]int{1, 1, 1, 1, 1, 1}, 8000, true},

		{[]int{2, 2, 2}, 200, true},
		{[]int{3, 3, 3}, 300, true},
		{[]int{5, 5, 5}, 500, true},
		{[]int{6, 6, 6}, 600, true},
		{[]int{6, 6, 6, 6}, 1200, true},
		{[]int{4, 4, 4, 4, 4, 4}, 3200, true},
		{[]int{2, 2, 2, 2, 2, 2}, 1600, true},

		{[]int{1, 2, 3, 4, 5, 6}, 1500, true},
		{[]int{2, 2, 3, 3, 4, 4}, 1500, true},
		{[]int{1, 1, 5, 5, 3, 3}, 1500, true},

		{[]int{2, 2, 2, 3, 3, 3}, 500, true},
		{[]int{1, 1, 1, 5}, 1050, true},
		{[]int{1, 1, 1, 1, 5, 5}, 2100, true},
		{[]int{5, 5, 5, 5}, 1000, true},

		// A single die that is not a 1 or 5 can never be set aside
		{[]int{2}, 0, false},
		{[]int{3}, 0, false},
		{[]int{1, 2}, 0, false},
		{[]int{3, 3}, 0, false},
		{[]int{2, 2, 3, 3}, 0, false},
		{[]int{}, 0, false},
	}

	for _, tc := range cases {
		score, ok := scoreSelection(tc.dice)
		if ok != tc.ok {
			t.Fatalf("scoreSelection(%v) ok = %v, want %v", tc.dice, ok, tc.ok)
		}
		if ok && score != tc.score {
			t.Fatalf("scoreSelection(%v) = %d, want %d", tc.dice, score, tc.score)
		}
	}
}

func TestRollHasScore(t *testing.T) {
	cases := []struct {
		dice []int
		want bool
	}{
		{[]int{2, 3, 4, 6, 2, 3}, false},
		{[]int{2, 2, 3, 3, 4, 6}, false},
		{[]int{3, 3, 4, 4, 6}, false},
		{[]int{2, 3, 4, 6}, false},

		{[]int{1, 2, 3, 4, 6, 6}, true},
		{[]int{2, 3, 4, 6, 6, 5}, true},
		{[]int{2, 3, 4, 6, 6, 6}, true},
		{[]int{2, 2, 3, 3, 4, 4}, true},
		{[]int{1, 2, 3, 4, 5, 6}, true},
	}

	for _, tc := range cases {
		if got := rollHasScore(tc.dice); got != tc.want {
			t.Fatalf("rollHasScore(%v) = %v, want %v", tc.dice, got, tc.want)
		}
	}
}

func TestBestSelection(t *testing.T) {
	cases := []struct {
		dice  []int
		score int
		used  int
	}{
		{[]int{1, 2, 3, 4, 6, 6}, 100, 1},
		{[]int{1, 1, 1, 2, 3, 4}, 1000, 3},
		{[]int{1, 2, 3, 4, 5, 6}, 1500, 6},
		{[]int{2, 2, 3, 3, 4, 4}, 1500, 6},
		{[]int{2, 3, 4, 6, 6, 5}, 50, 1},
		{[]int{2, 3, 4, 6, 2, 3}, 0, 0},
	}

	for _, tc := range cases {
		mask, score := bestSelection(tc.dice)
		if score != tc.score {
			t.Fatalf("bestSelection(%v) score = %d, want %d", tc.dice, score, tc.score)
		}
		used := 0
		for _, k := range mask {
			if k {
				used++
			}
		}
		if used != tc.used {
			t.Fatalf("bestSelection(%v) used %d dice, want %d", tc.dice, used, tc.used)
		}
	}
}

// Every subset chosen by bestSelection must itself be a legal set-aside.
func TestBestSelectionAlwaysLegal(t *testing.T) {
	var walk func(dice []int, n int)
	walk = func(dice []int, n int) {
		if n == 0 {
			mask, score := bestSelection(dice)
			kept := []int{}
			for i, k := range mask {
				if k {
					kept = append(kept, dice[i])
				}
			}
			if score == 0 {
				if rollHasScore(dice) {
					t.Fatalf("bestSelection(%v) found nothing but roll scores", dice)
				}
				return
			}
			got, ok := scoreSelection(kept)
			if !ok || got != score {
				t.Fatalf("bestSelection(%v) picked illegal subset %v", dice, kept)
			}
			return
		}
		for f := 1; f <= 6; f++ {
			walk(append(dice, f), n-1)
		}
	}
	walk([]int{}, 5)
}

// Every die the server marks selectable must, taken together, form a legal
// set-aside. Clients rely on this to offer a one key "keep everything that
// scores" without implementing any scoring rules of their own.
func TestSelectableUnionIsAlwaysLegal(t *testing.T) {
	var walk func(dice []int, n int)
	walk = func(dice []int, n int) {
		if n == 0 {
			mask := selectableMask(dice)

			kept := []int{}
			for i := 0; i < len(mask); i++ {
				if mask[i] == '1' {
					kept = append(kept, dice[i])
				}
			}

			if len(kept) == 0 {
				if rollHasScore(dice) {
					t.Fatalf("roll %v scores but nothing was marked selectable", dice)
				}
				return
			}

			if _, ok := scoreSelection(kept); !ok {
				t.Fatalf("roll %v marked %v selectable, which is not a legal set-aside", dice, kept)
			}
			return
		}
		for f := 1; f <= 6; f++ {
			walk(append(dice, f), n-1)
		}
	}

	for size := 1; size <= NUM_DICE; size++ {
		walk([]int{}, size)
	}
}

func TestApplyKeepMask(t *testing.T) {
	dice := []int{1, 2, 3, 4, 5, 6}

	kept, pool, ok := applyKeepMask(dice, "100010")
	if !ok {
		t.Fatal("expected mask to apply")
	}
	if diceToString(kept) != "15" {
		t.Fatalf("kept = %v, want [1 5]", kept)
	}
	if diceToString(pool) != "2346" {
		t.Fatalf("pool = %v, want [2 3 4 6]", pool)
	}

	if _, _, ok := applyKeepMask(dice, "1000"); ok {
		t.Fatal("expected wrong-length mask to fail")
	}
	if _, _, ok := applyKeepMask(dice, "10002x"); ok {
		t.Fatal("expected invalid mask character to fail")
	}
}

func TestParseDiceRoundTrip(t *testing.T) {
	if got := diceToString(parseDice("135524")); got != "135524" {
		t.Fatalf("round trip = %q", got)
	}
	if got := len(parseDice("1x3")); got != 2 {
		t.Fatalf("expected junk to be skipped, got %d dice", got)
	}
}
