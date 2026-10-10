package notifier

import (
	"strings"
	"testing"
	"time"
)

func TestFormatTrafficReportMessageSeparatesNodes(t *testing.T) {
	start := time.Date(2026, 9, 25, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 0, 1).Add(-time.Nanosecond)
	got := formatTrafficReportMessage("daily", start.UTC(), end.UTC(), []trafficReportLine{
		{Name: "小节点", Used: 100 * 1024 * 1024},
		{Name: "大节点", Used: 2 * 1024 * 1024 * 1024},
	})
	if !strings.Contains(got, "📅 2026-09-25") {
		t.Fatalf("period line missing: %q", got)
	}
	if !strings.Contains(got, "大节点\n2.00 GB\n\n小节点\n100.00 MB") {
		t.Fatalf("nodes are not separated and ordered by usage: %q", got)
	}
	if !strings.Contains(got, "合计  2 台") {
		t.Fatalf("total line missing: %q", got)
	}
	if strings.Contains(got, "昨日流量") {
		t.Fatalf("old dense suffix still present: %q", got)
	}
}
