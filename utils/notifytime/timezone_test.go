package notifytime

import (
	"testing"
	"time"
)

func TestNormalizeAcceptsOffsetsAndIANA(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"", Default},
		{"  utc+8 ", "UTC+8"},
		{"+08:00", "UTC+8"},
		{"UTC+08:00", "UTC+8"},
		{"GMT-5", "UTC-5"},
		{"UTC+5:30", "UTC+5:30"},
		{"UTC", "UTC"},
		{"Asia/Shanghai", "Asia/Shanghai"},
	} {
		got, err := Normalize(tc.in)
		if err != nil {
			t.Fatalf("Normalize(%q) error = %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeRejectsUnknownTimezone(t *testing.T) {
	for _, in := range []string{"Local", "Not/AZone", "UTC+99", "UTC+8:99"} {
		if _, err := Normalize(in); err == nil {
			t.Fatalf("Normalize(%q) accepted an invalid timezone", in)
		}
	}
}

func TestFormatInWritesOffset(t *testing.T) {
	instant := time.Date(2026, 9, 26, 18, 18, 28, 0, time.UTC)
	got := FormatIn(instant, time.FixedZone("UTC+8", 8*60*60))
	if got != "2026-09-27 02:18:28 UTC+8" {
		t.Fatalf("formatted = %q", got)
	}
	got = FormatIn(instant, time.FixedZone("UTC+5:30", 5*3600+30*60))
	if got != "2026-09-26 23:48:28 UTC+5:30" {
		t.Fatalf("half-hour formatted = %q", got)
	}
}

func TestDefaultFormatIgnoresProcessLocalZone(t *testing.T) {
	original := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = original })

	instant := time.Date(2026, 7, 17, 1, 30, 0, 0, time.UTC)
	got := Format(instant)
	if got != "2026-07-17 09:30:00 UTC+8" {
		t.Fatalf("formatted = %q", got)
	}
}
