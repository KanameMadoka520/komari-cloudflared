package notifier

import (
	"fmt"
	"strings"

	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/utils/notifytmpl"
)

func noticeTitle(id string, data map[string]string, nodes []map[string]string) string {
	title, _, err := notifytmpl.Render(id, data, nodes)
	if err != nil {
		return ""
	}
	return title
}

func noticeBody(id string, data map[string]string, nodes []map[string]string) string {
	_, body, err := notifytmpl.Render(id, data, nodes)
	if err != nil {
		return ""
	}
	return body
}

func clientDisplayName(client models.Client) string {
	name := strings.TrimSpace(client.Name)
	if name == "" {
		return client.UUID
	}
	return name
}

func trafficTypeLabel(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "up":
		return "仅上传"
	case "down":
		return "仅下载"
	case "max":
		return "取较大方向"
	case "min":
		return "取较小方向"
	default:
		return "上传+下载"
	}
}

func loadAlertData(task models.LoadNotification) map[string]string {
	return map[string]string{
		"rule":      strings.TrimSpace(task.Name),
		"metric":    metricLabel(task.Metric),
		"threshold": fmt.Sprintf("%.0f", task.Threshold),
		"ratio":     fmt.Sprintf("%.0f%%", task.Ratio*100),
	}
}

func loadAlertNodes(names []string) []map[string]string {
	nodes := make([]map[string]string, 0, len(names))
	for _, name := range names {
		nodes = append(nodes, map[string]string{"name": name})
	}
	return nodes
}

func metricLabel(metric string) string {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "cpu":
		return "CPU"
	case "ram":
		return "内存"
	case "swap":
		return "交换区"
	case "disk":
		return "磁盘"
	case "load":
		return "负载"
	case "temp":
		return "温度"
	case "gpu":
		return "GPU"
	case "net_in", "netin":
		return "入站"
	case "net_out", "netout":
		return "出站"
	default:
		if metric == "" {
			return "指标"
		}
		return metric
	}
}
