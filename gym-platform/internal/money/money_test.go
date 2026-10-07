package money

import "testing"

func TestParseNaira(t *testing.T) {
	good := map[string]int64{
		"15000":     1500000,
		"15,000.50": 1500050,
		"₦200":      20000,
		"0.5":       50,
		".5":        50,
		"0":         0,
	}
	for in, want := range good {
		got, err := ParseNaira(in)
		if err != nil || got != want {
			t.Errorf("ParseNaira(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "-5", "12.345", "1.", "1e3"} {
		if _, err := ParseNaira(in); err == nil {
			t.Errorf("ParseNaira(%q) should fail", in)
		}
	}
}

func TestFormat(t *testing.T) {
	if got := FormatNaira(1500050); got != "₦15,000.50" {
		t.Errorf("FormatNaira = %q", got)
	}
	if got := FormatNaira(99); got != "₦0.99" {
		t.Errorf("FormatNaira = %q", got)
	}
	if got := FormatPlain(1500000); got != "15000" {
		t.Errorf("FormatPlain = %q", got)
	}
}