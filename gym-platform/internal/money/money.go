package money

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidAmount = errors.New("Enter a valid amount, e.g. 15000 or 15000.50.")

// ParseNaira turns "15,000.50" into kobo (1500050) without using floats.
func ParseNaira(s string) (int64, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "₦")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, ErrInvalidAmount
	}

	whole, frac, hasDot := strings.Cut(s, ".")
	if whole == "" {
		whole = "0"
	}
	if !allDigits(whole) || len(whole) > 12 {
		return 0, ErrInvalidAmount
	}
	if hasDot {
		if frac == "" || len(frac) > 2 || !allDigits(frac) {
			return 0, ErrInvalidAmount
		}
		frac = (frac + "0")[:2]
	} else {
		frac = "00"
	}

	w, _ := strconv.ParseInt(whole, 10, 64)
	f, _ := strconv.ParseInt(frac, 10, 64)
	return w*100 + f, nil
}

// FormatNaira renders kobo for display: 1500050 -> "₦15,000.50".
func FormatNaira(kobo int64) string {
	return fmt.Sprintf("₦%s.%02d", withCommas(kobo/100), kobo%100)
}

// FormatPlain renders kobo for an input box: 1500000 -> "15000", 1500050 -> "15000.50".
func FormatPlain(kobo int64) string {
	if kobo%100 == 0 {
		return strconv.FormatInt(kobo/100, 10)
	}
	return fmt.Sprintf("%d.%02d", kobo/100, kobo%100)
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func withCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	pre := len(s) % 3
	if pre > 0 {
		b.WriteString(s[:pre])
	}
	for i := pre; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
