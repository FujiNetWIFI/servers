package main

import (
	"testing"
)

func testSlice(a, b []*string) bool {
	if len(a) != len(b) {
		return false
	}

	for i, p := range a {
		if p != b[i] {
			return false
		}
	}
	return true
}

func TestBuffer(t *testing.T) {
	buff := newRingBuffer(5)
	var a, b, c, d, e, f, g string
	a = "1"
	b = "2"
	c = "3"
	d = "4"
	e = "5"
	f = "6"
	g = "7"

	var expected = []struct {
		p *string
		b []*string
	}{
		{&a, []*string{nil, nil, nil, nil, &a}},
		{&b, []*string{nil, nil, nil, &a, &b}},
		{&c, []*string{nil, nil, &a, &b, &c}},
		{&d, []*string{nil, &a, &b, &c, &d}},
		{&e, []*string{&a, &b, &c, &d, &e}},
		{&f, []*string{&b, &c, &d, &e, &f}},
		{&g, []*string{&c, &d, &e, &f, &g}},
	}

	for _, test := range expected {
		buff.add(test.p)
		res := buff.readAll()
		if !testSlice(test.b, res) {
			t.Errorf("expected: %v, got %v", test.b, res)
		}

	}
}
