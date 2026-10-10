package messageSender

import (
	"fmt"
	"strings"

	"time"

	"github.com/komari-monitor/komari/database/models"
	messageevent "github.com/komari-monitor/komari/database/models/messageEvent"
	"github.com/komari-monitor/komari/internal/config"
)

// DefaultNotificationTemplate 是通知正文模板。
// 标题由发送渠道单独展示，正文只保留内容和时间，避免同一条消息里重复出现事件名和节点名单。
const DefaultNotificationTemplate = "{{message}}\n\n🕐 {{time}}"

func notificationTitle(event models.EventMessage) string {
	if title := strings.TrimSpace(event.Title); title != "" {
		return title
	}
	label := messageevent.Label(event.Event)
	emoji := strings.TrimSpace(fmt.Sprint(event.Emoji))
	if emoji == "" || emoji == "<nil>" {
		return label
	}
	return emoji + " " + label
}

// Compose 套用当前通知外壳模板，返回发出去的正文。标题由调用方单独展示。
func Compose(event models.EventMessage) (string, error) {
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	} else {
		event.Time = event.Time.UTC()
	}
	template := DefaultNotificationTemplate
	if config.Available() {
		if stored, err := config.GetAs[string](config.NotificationTemplateKey, DefaultNotificationTemplate); err == nil && strings.TrimSpace(stored) != "" {
			template = stored
		}
	}
	return parseTemplate(template, event), nil
}
