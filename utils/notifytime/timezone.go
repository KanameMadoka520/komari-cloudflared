package notifytime

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/komari-monitor/komari/internal/config"
)

const (
	// Default 是通知时间的默认时区，全年 UTC+8。
	Default = "Asia/Shanghai"
	maxName = 80
)

var (
	errInvalid = fmt.Errorf("invalid notification timezone")
	offsetRE   = regexp.MustCompile(`(?i)^(?:UTC|GMT)?([+-])(\d{1,2})(?::?(\d{2}))?$`)
)

// Normalize 校验并整理管理员保存的时区。空值回到默认的 Asia/Shanghai。
// 偏移写法统一成 UTC+8、UTC-5:30 这种形式；IANA 名称保持原样。
func Normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Default, nil
	}
	if len(raw) > maxName {
		return "", errInvalid
	}
	if canon, ok := canonicalOffset(raw); ok {
		return canon, nil
	}
	if strings.EqualFold(raw, "UTC") || strings.EqualFold(raw, "GMT") || raw == "Z" {
		return "UTC", nil
	}
	if strings.EqualFold(raw, "Local") {
		return "", errInvalid
	}
	loc, err := time.LoadLocation(raw)
	if err != nil || loc == nil {
		return "", errInvalid
	}
	return loc.String(), nil
}

// Parse 把已保存的时区名称解析成地点。无法识别时返回错误。
func Parse(name string) (*time.Location, error) {
	canon, err := Normalize(name)
	if err != nil {
		return nil, err
	}
	if canon == "UTC" {
		return time.UTC, nil
	}
	if canon == Default {
		if loc, err := time.LoadLocation(Default); err == nil {
			return loc, nil
		}
		return time.FixedZone("UTC+8", 8*60*60), nil
	}
	if offset, ok := offsetSeconds(canon); ok {
		return time.FixedZone(canon, offset), nil
	}
	return time.LoadLocation(canon)
}

// Location 是当前通知时区。没有配置或配置无法识别时使用 UTC+8。
func Location() *time.Location {
	name := Default
	if config.Available() {
		stored, err := config.GetAs[string](config.NotificationTimezoneKey, Default)
		if err == nil {
			if trimmed := strings.TrimSpace(stored); trimmed != "" {
				name = trimmed
			}
		}
	}
	loc, err := Parse(name)
	if err != nil || loc == nil {
		loc, _ = Parse(Default)
	}
	if loc == nil {
		return time.FixedZone("UTC+8", 8*60*60)
	}
	return loc
}

// Format 按当前通知时区格式化时间，并在末尾写上 UTC 偏移。
func Format(t time.Time) string {
	return FormatIn(t, Location())
}

// FormatIn 按指定地点格式化时间，并写上该时刻的 UTC 偏移。
func FormatIn(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = time.FixedZone("UTC+8", 8*60*60)
	}
	if t.IsZero() {
		t = time.Now()
	}
	shown := t.In(loc)
	return shown.Format("2006-01-02 15:04:05") + " " + OffsetLabel(shown)
}

// OffsetLabel 把某一时刻的偏移写成 UTC+8 或 UTC-5:30。
func OffsetLabel(t time.Time) string {
	_, offset := t.Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	mins := (offset % 3600) / 60
	if mins == 0 {
		return fmt.Sprintf("UTC%s%d", sign, hours)
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, hours, mins)
}

func canonicalOffset(raw string) (string, bool) {
	compact := strings.ReplaceAll(strings.TrimSpace(raw), " ", "")
	match := offsetRE.FindStringSubmatch(compact)
	if match == nil {
		return "", false
	}
	sign := match[1]
	var hours, mins int
	if _, err := fmt.Sscanf(match[2], "%d", &hours); err != nil {
		return "", false
	}
	if match[3] != "" {
		if _, err := fmt.Sscanf(match[3], "%d", &mins); err != nil {
			return "", false
		}
	}
	if hours > 14 || mins > 59 || (hours == 14 && mins != 0) {
		return "", false
	}
	if mins == 0 {
		return fmt.Sprintf("UTC%s%d", sign, hours), true
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, hours, mins), true
}

func offsetSeconds(canon string) (int, bool) {
	match := offsetRE.FindStringSubmatch(canon)
	if match == nil {
		return 0, false
	}
	var hours, mins int
	fmt.Sscanf(match[2], "%d", &hours)
	if match[3] != "" {
		fmt.Sscanf(match[3], "%d", &mins)
	}
	seconds := hours*3600 + mins*60
	if match[1] == "-" {
		seconds = -seconds
	}
	return seconds, true
}
