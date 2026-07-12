package main

import (
	"reflect"
	"testing"

	"github.com/botforge-pro/sf2subset/internal/sf2"
)

func TestParseSelectors(t *testing.T) {
	cases := []struct {
		in   string
		want []sf2.Selector
	}{
		{"0,18,81", []sf2.Selector{{Program: 0}, {Program: 18}, {Program: 81}}},
		{"0:0, 0:18 ,128:38", []sf2.Selector{{Bank: 0, Program: 0}, {Bank: 0, Program: 18}, {Bank: 128, Program: 38}}},
		{"18", []sf2.Selector{{Program: 18}}},
	}
	for _, c := range cases {
		got, err := parseSelectors(c.in)
		if err != nil {
			t.Errorf("parseSelectors(%q): %v", c.in, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseSelectors(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseSelectorsErrors(t *testing.T) {
	for _, in := range []string{"", "  ", "x", "0:", ":5", "0:x"} {
		if _, err := parseSelectors(in); err == nil {
			t.Errorf("parseSelectors(%q): expected error", in)
		}
	}
}
