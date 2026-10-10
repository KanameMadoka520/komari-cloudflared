package messageSender

import (
	"testing"
	"time"

	"github.com/komari-monitor/komari/database/models"
)

func TestParseTemplateFormatsEventTimeInNotificationTimezone(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = originalLocal })

	eventTime := time.Date(2026, 7, 17, 1, 30, 0, 123456789, time.UTC)
	got := parseTemplate("{{time}}", models.EventMessage{Time: eventTime})
	want := "2026-07-17 09:30:00 UTC+8"
	if got != want {
		t.Fatalf("formatted event time = %q, want %q", got, want)
	}
}

func TestParseTemplateFormatsAnyTypedEventField(t *testing.T) {
	for _, tc := range []struct {
		event any
		want  string
	}{
		{event: models.EventMessage{Event: "Offline"}, want: "节点离线"},
		{event: models.EventMessage{Event: 42}, want: "42"},
		{event: models.EventMessage{Event: nil}, want: ""},
	} {
		if got := parseTemplate("{{event}}", tc.event); got != tc.want {
			t.Fatalf("parseTemplate with Event=%v = %q, want %q", tc.event, got, tc.want)
		}
	}
}
