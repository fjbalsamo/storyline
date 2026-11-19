package trace

import (
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	idx, err := Scan("testdata/repo")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		id   string
		want string // comma-separated refs; "" = uncovered
	}{
		{"AC-1.1", "go/booking_test.go:7"},                            // Go; production code, node_modules, storyline, vendor, target ignored
		{"AC-1.2", "web/__tests__/helper.js:4, web/cancel.test.ts:2"}, // __tests__ dir and .test.ts
		{"AC-1.3", "py/booking_test.py:1, py/test_booking.py:2"},      // both Python patterns, any case
		{"AC-1.4", ""},                         // appears only in comments (//, #, /* */, *)
		{"AC-1.5", "rust/tests/cli.rs:2"},      // Rust under tests/
		{"AC-1.6", "rust/src/lib.rs:6"},        // Rust with #[cfg(test)]
		{"AC-1.7", ""},                         // non-test Rust file
		{"AC-1.8", "java/BookingTest.java:2"},  // Java *Test
		{"AC-1.9", "kotlin/BookingTests.kt:2"}, // Kotlin *Tests
		{"AC-1.10", "web/legacy.spec.js:1"},    // AC-1.1 must not match AC-1.10 and vice versa
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			c, ok := ParseCriterion(tt.id)
			if !ok {
				t.Fatal("bad id")
			}
			var got []string
			for _, r := range idx[c] {
				got = append(got, r.String())
			}
			if strings.Join(got, ", ") != tt.want {
				t.Errorf("refs = %q, want %q", strings.Join(got, ", "), tt.want)
			}
		})
	}
}

func TestRefBoundaries(t *testing.T) {
	tests := []struct {
		line string
		want []Criterion
	}{
		{"AC-1.1", []Criterion{{1, 1}}},
		{"AC-1.1.", []Criterion{{1, 1}}},
		{"ac_12_1_foo", []Criterion{{12, 1}}},
		{"AC-1.10", []Criterion{{1, 10}}},
		{"AC-1.1x", nil},
		{"AC-01.01", []Criterion{{1, 1}}},
		{"AC-1.1,AC-1.2", []Criterion{{1, 1}, {1, 2}}},
	}
	for _, tt := range tests {
		var got []Criterion
		for _, m := range refRe.FindAllStringSubmatch(tt.line, -1) {
			c, _ := toCriterion(m[1], m[2])
			got = append(got, c)
		}
		if len(got) != len(tt.want) {
			t.Errorf("%q: got %v, want %v", tt.line, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("%q: got %v, want %v", tt.line, got, tt.want)
			}
		}
	}
}

func TestIsComment(t *testing.T) {
	for line, want := range map[string]bool{
		"  // AC-1.1": true, "# AC-1.1": true, "/* AC-1.1 */": true, " * AC-1.1": true, "-- AC-1.1": true,
		"it('AC-1.1')": false, "x := 1 // AC-1.1": false,
	} {
		if got := isComment(line); got != want {
			t.Errorf("isComment(%q) = %v, want %v", line, got, want)
		}
	}
}
