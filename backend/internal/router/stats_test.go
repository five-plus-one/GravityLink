package router

import (
	"testing"
	"time"
)

func TestVisitorTimeBoundaries(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("Asia/Shanghai", 8*3600)
	defer func() { time.Local = original }()
	start, err := parseVisitorTime("2026-09-13", false)
	if err != nil || start.Hour() != 0 {
		t.Fatal(start, err)
	}
	end, err := parseVisitorTime("2026-09-13", true)
	if err != nil || end.Sub(start) != 24*time.Hour {
		t.Fatal(end, err)
	}
	instant, err := parseVisitorTime("2026-09-12T16:30:45.123Z", false)
	if err != nil || instant.Day() != 13 || instant.Hour() != 0 || instant.Minute() != 30 || instant.Nanosecond() != 123000000 {
		t.Fatal(instant, err)
	}
	for _, v := range []string{"invalid", "2026-02-30", "2026-09-13T10:00:00"} {
		if _, err := parseVisitorTime(v, false); err == nil {
			t.Fatalf("accepted %s", v)
		}
	}
}
