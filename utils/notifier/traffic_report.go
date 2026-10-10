package notifier

import (
	"fmt"
	"sort"
	"strings"
	"time"

	logger "github.com/komari-monitor/komari/utils/log"

	"context"

	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	messageevent "github.com/komari-monitor/komari/database/models/messageEvent"
	"github.com/komari-monitor/komari/internal/config"
	"github.com/komari-monitor/komari/internal/metricstore"
	"github.com/komari-monitor/komari/internal/scheduler"
	"github.com/komari-monitor/komari/utils/messageSender"
	"github.com/komari-monitor/komari/utils/notifytmpl"
)

// InitTrafficReportSchedule 注册三个定时任务：日报、周报、月报
func InitTrafficReportSchedule() {
	// 日报：每天凌晨 0 点
	if err := scheduler.AddFunc("traffic-report-daily", "0 0 0 * * *", func() {
		sendTrafficReport(true, false, false)
	}); err != nil {
		logger.ErrorArgs("notifier", "Failed to register daily traffic report job:", err)
	}

	// 周报：每周一凌晨 0 点 (dow=1)
	if err := scheduler.AddFunc("traffic-report-weekly", "0 0 0 * * 1", func() {
		sendTrafficReport(false, true, false)
	}); err != nil {
		logger.ErrorArgs("notifier", "Failed to register weekly traffic report job:", err)
	}

	// 月报：每月 1 日凌晨 0 点
	if err := scheduler.AddFunc("traffic-report-monthly", "0 0 0 1 * *", func() {
		sendTrafficReport(false, false, true)
	}); err != nil {
		logger.ErrorArgs("notifier", "Failed to register monthly traffic report job:", err)
	}
}

// sendTrafficReport 汇聚所有启用了指定报告类型的服务器流量，合并成一条通知发送
func sendTrafficReport(daily, weekly, monthly bool) {
	// 检查全局通知开关
	enabled, err := config.GetAs[bool](config.NotificationEnabledKey, false)
	if err != nil || !enabled {
		return
	}

	db := dbcore.GetDBInstance()
	now := time.Now().UTC()

	var eventType, label string

	switch {
	case daily:
		eventType = messageevent.DReport
		label = "daily"
	case weekly:
		eventType = messageevent.WReport
		label = "weekly"
	case monthly:
		eventType = messageevent.MReport
		label = "monthly"
	default:
		return
	}
	start, end := previousTrafficReportRange(now, label)

	// 查询所有启用该类型报告的服务器配置
	var notifications []models.TrafficReportNotification
	query := db.Model(&models.TrafficReportNotification{}).Where("enable = ?", true)
	if daily {
		query = query.Where("daily = ?", true)
	} else if weekly {
		query = query.Where("weekly = ?", true)
	} else if monthly {
		query = query.Where("monthly = ?", true)
	}
	if err := query.Find(&notifications).Error; err != nil {
		logger.Errorf("notifier", "Failed to query traffic report notifications (%s): %v", label, err)
		return
	}
	if len(notifications) == 0 {
		return
	}

	// 获取客户端信息
	clientUUIDs := make([]string, 0, len(notifications))
	for _, n := range notifications {
		clientUUIDs = append(clientUUIDs, n.Client)
	}
	var clientList []models.Client
	if err := db.Where("uuid IN ?", clientUUIDs).Find(&clientList).Error; err != nil {
		logger.Errorf("notifier", "Failed to query clients for traffic report (%s): %v", label, err)
		return
	}
	clientMap := make(map[string]models.Client, len(clientList))
	for _, c := range clientList {
		clientMap[c.UUID] = c
	}

	reportLines := make([]trafficReportLine, 0, len(notifications))
	eventClients := make([]models.Client, 0, len(notifications))
	for _, n := range notifications {
		c, ok := clientMap[n.Client]
		if !ok {
			continue
		}

		used, err := getClientTrafficInRange(n.Client, c.TrafficLimitType, start, end)
		if err != nil {
			logger.Errorf("notifier", "Failed to compute traffic for client %s (%s): %v", n.Client, label, err)
			continue
		}

		name := strings.TrimSpace(c.Name)
		if name == "" {
			name = c.UUID
		}
		reportLines = append(reportLines, trafficReportLine{Name: name, Used: used})
		eventClients = append(eventClients, c)
	}

	if len(reportLines) == 0 {
		return
	}

	title, message := formatTrafficReport(label, start, end, reportLines)

	if err := messageSender.SendNotification(models.EventMessage{
		Event:   eventType,
		Clients: eventClients,
		Time:    now,
		Title:   title,
		Message: message,
	}); err != nil {
		logger.Errorf("notifier", "Failed to send %s traffic report: %v", label, err)
	}
}

type trafficReportLine struct {
	Name string
	Used int64
}

func formatTrafficReportMessage(period string, start, end time.Time, lines []trafficReportLine) string {
	_, body := formatTrafficReport(period, start, end, lines)
	return body
}

func formatTrafficReport(period string, start, end time.Time, lines []trafficReportLine) (string, string) {
	sorted := append([]trafficReportLine(nil), lines...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Used == sorted[j].Used {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].Used > sorted[j].Used
	})
	localStart := start.In(time.Local)
	localEnd := end.In(time.Local)
	var periodText string
	id := "traffic_daily"
	switch period {
	case "weekly":
		id = "traffic_weekly"
		periodText = localStart.Format("2006-01-02") + "  →  " + localEnd.Format("2006-01-02")
	case "monthly":
		id = "traffic_monthly"
		periodText = localStart.Format("2006年1月")
	default:
		periodText = localStart.Format("2006-01-02")
	}
	nodes := make([]map[string]string, 0, len(sorted))
	var total int64
	for _, line := range sorted {
		total += line.Used
		nodes = append(nodes, map[string]string{"name": line.Name, "used": humanBytes(line.Used)})
	}
	title, body, err := notifytmpl.Render(id, map[string]string{
		"period": periodText,
		"count":  fmt.Sprintf("%d", len(sorted)),
		"total":  humanBytes(total),
	}, nodes)
	if err != nil {
		return "", ""
	}
	return title, body
}

func previousTrafficReportRange(now time.Time, period string) (time.Time, time.Time) {
	localNow := now.In(time.Local)
	var startLocal, endLocal time.Time

	switch period {
	case "daily":
		yesterday := localNow.AddDate(0, 0, -1)
		startLocal = time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.Local)
		endLocal = startLocal.AddDate(0, 0, 1)
	case "weekly":
		weekday := int(localNow.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		lastMonday := localNow.AddDate(0, 0, -(weekday-1)-7)
		startLocal = time.Date(lastMonday.Year(), lastMonday.Month(), lastMonday.Day(), 0, 0, 0, 0, time.Local)
		endLocal = startLocal.AddDate(0, 0, 7)
	case "monthly":
		endLocal = time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, time.Local)
		startLocal = endLocal.AddDate(0, -1, 0)
	default:
		return time.Time{}, time.Time{}
	}

	return startLocal.UTC(), endLocal.Add(-time.Nanosecond).UTC()
}

// getClientTrafficInRange 查询某客户端在指定时间段内的流量增量。
//
// 历史监控数据已完全迁移到 metric store，这里从 metric store 读取区间内记录并
// 累加精确的流量增量字段计算用量；缺失增量时回退到累计流量差值。
func getClientTrafficInRange(clientUUID string, trafficType string, start, end time.Time) (int64, error) {
	ctx := context.Background()
	recs, err := metricstore.GetRecordsByClientAndTime(ctx, clientUUID, start, end)
	if err != nil {
		return 0, err
	}

	records := make([]trafficDeltaRecord, 0, len(recs))
	for _, r := range recs {
		records = append(records, trafficDeltaRecord{
			Time:         r.Time,
			NetTotalUp:   r.NetTotalUp,
			NetTotalDown: r.NetTotalDown,
			TrafficUp:    r.TrafficUp,
			TrafficDown:  r.TrafficDown,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].Time.Before(records[j].Time)
	})

	// 计算增量基线（区间开始前最后一条累计流量）
	var previous *trafficDeltaRecord
	baseline, err := metricstore.GetLatestTrafficBefore(ctx, []string{clientUUID}, start)
	if err != nil {
		return 0, err
	}
	if base, ok := baseline[clientUUID]; ok {
		previous = &trafficDeltaRecord{
			Time:         base.Time,
			NetTotalUp:   base.NetTotalUp,
			NetTotalDown: base.NetTotalDown,
		}
	}

	totalUp, totalDown := sumTrafficDeltas(records, previous)
	return computeUsedByType(strings.ToLower(trafficType), totalUp, totalDown), nil
}

type trafficDeltaRecord struct {
	Time         time.Time
	NetTotalUp   int64
	NetTotalDown int64
	TrafficUp    int64
	TrafficDown  int64
}

func sumTrafficDeltas(records []trafficDeltaRecord, previous *trafficDeltaRecord) (int64, int64) {
	var totalUp int64
	var totalDown int64

	for i := range records {
		up := records[i].TrafficUp
		down := records[i].TrafficDown

		if previous != nil {
			up = trafficDeltaOrFallback(up, records[i].NetTotalUp, previous.NetTotalUp)
			down = trafficDeltaOrFallback(down, records[i].NetTotalDown, previous.NetTotalDown)
		}
		totalUp += up
		totalDown += down
		previous = &records[i]
	}

	return totalUp, totalDown
}

func trafficDeltaOrFallback(storedDelta, currentTotal, previousTotal int64) int64 {
	if storedDelta > 0 {
		return storedDelta
	}
	return metricstore.TrafficCounterDelta(currentTotal, previousTotal)
}
