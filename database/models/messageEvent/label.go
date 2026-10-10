package messageevent

import (
	"fmt"
	"strings"
)

// Label 把事件代码换成通知标题里使用的中文名称。
func Label(event any) string {
	switch strings.TrimSpace(fmt.Sprint(event)) {
	case Offline:
		return "节点离线"
	case Online:
		return "节点上线"
	case Expire:
		return "到期提醒"
	case Renew:
		return "自动续期"
	case Login:
		return "登录提醒"
	case Alert:
		return "负载告警"
	case Traffic:
		return "流量提醒"
	case DReport:
		return "流量日报"
	case WReport:
		return "流量周报"
	case MReport:
		return "流量月报"
	case "Test":
		return "测试消息"
	default:
		text := strings.TrimSpace(fmt.Sprint(event))
		if text == "" || text == "<nil>" {
			return "通知"
		}
		return text
	}
}
