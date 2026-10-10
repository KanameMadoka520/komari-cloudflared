package notifytmpl

import (
	"strings"
	"testing"
)

func TestRenderDailyReportUsesSampleLayout(t *testing.T) {
	title, body, err := Render("traffic_daily", map[string]string{
		"period": "2026-09-25",
		"count":  "2",
		"total":  "16.68 GB",
	}, []map[string]string{
		{"name": "大节点", "used": "14.82 GB"},
		{"name": "小节点", "used": "1.86 GB"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if title != "📊 流量日报" {
		t.Fatalf("title = %q", title)
	}
	if !strings.Contains(body, "📅 2026-09-25") || !strings.Contains(body, "大节点\n14.82 GB\n\n小节点\n1.86 GB") {
		t.Fatalf("body = %q", body)
	}
	if !strings.Contains(body, "合计  2 台  ·  16.68 GB") {
		t.Fatalf("total missing: %q", body)
	}
}

func TestPreviewDraftUsesEditedText(t *testing.T) {
	preview, err := PreviewDraft("offline", "离线", "节点 {{name}}", "")
	if err != nil {
		t.Fatal(err)
	}
	if preview.Title != "离线" {
		t.Fatalf("title = %q", preview.Title)
	}
	if !strings.Contains(preview.Text, "节点 netcup RS-x社区+各种服务基建") {
		t.Fatalf("preview = %q", preview.Text)
	}
	if !strings.Contains(preview.Text, "🕐 2026-09-26 00:00:00 UTC+8") {
		t.Fatalf("time line missing: %q", preview.Text)
	}
}
