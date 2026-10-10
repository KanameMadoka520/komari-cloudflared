package notifytmpl

import (
	"fmt"
	"strings"
	"time"

	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/internal/config"
	"github.com/komari-monitor/komari/utils/messageSender"
	"github.com/komari-monitor/komari/utils/notifytime"
)

const (
	maxTitleLen = 200
	maxBodyLen  = 8000
	maxNodeLen  = 4000
)

// Definition 是一种通知的内置模板。
type Definition struct {
	ID               string
	Title            string
	Body             string
	Node             string
	NodeJoin         string
	Placeholders     []string
	NodePlaceholders []string
	Sample           map[string]string
	SampleNodes      []map[string]string
}

// Stored 是管理员保存的覆盖。空字符串表示这一段留空，不会回退到默认值。
type Stored struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Node  string `json:"node"`
}

// View 是后台列表和编辑器使用的一条模板。
type View struct {
	ID               string              `json:"id"`
	Title            string              `json:"title"`
	Body             string              `json:"body"`
	Node             string              `json:"node"`
	HasNode          bool                `json:"has_node"`
	Customized       bool                `json:"customized"`
	Placeholders     []string            `json:"placeholders"`
	NodePlaceholders []string            `json:"node_placeholders"`
	Preview          Preview             `json:"preview"`
	Sample           map[string]string   `json:"sample"`
	SampleNodes      []map[string]string `json:"sample_nodes,omitempty"`
}

// Preview 是按样例数据渲染后的标题和完整文本。
type Preview struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func Catalog() []Definition {
	return []Definition{
		report("traffic_daily", "📊 流量日报"),
		report("traffic_weekly", "📈 流量周报"),
		report("traffic_monthly", "📅 流量月报"),
		{
			ID: "offline", Title: "🔴 节点离线", Body: "{{name}}",
			Placeholders: []string{"name"},
			Sample:       map[string]string{"name": "netcup RS-x社区+各种服务基建"},
		},
		{
			ID: "online", Title: "🟢 节点上线", Body: "{{name}}",
			Placeholders: []string{"name"},
			Sample:       map[string]string{"name": "netcup RS-x社区+各种服务基建"},
		},
		{
			ID: "traffic_alert", Title: "⚠️ 流量提醒",
			Body:         "{{name}}\n\n已用  {{percent}}%\n{{used}}  /  {{limit}}\n计费  {{billing}}",
			Placeholders: []string{"name", "percent", "used", "limit", "billing"},
			Sample: map[string]string{
				"name": "OVHCloud 塔菲API源站机 美西", "percent": "80",
				"used": "8.00 GB", "limit": "10.00 GB", "billing": "上传+下载",
			},
		},
		{
			ID: "expire", Title: "⏳ 到期提醒", Body: "{{nodes}}",
			Node: "{{name}}\n还剩  {{days}} 天", NodeJoin: "\n\n",
			Placeholders: []string{"nodes"}, NodePlaceholders: []string{"name", "days"},
			SampleNodes: []map[string]string{
				{"name": "日本东京精品", "days": "3"},
				{"name": "英国伦敦CN2", "days": "6"},
			},
		},
		{
			ID: "renew", Title: "🔄 自动续期",
			Body:         "{{name}}\n续期至  {{until}}",
			Placeholders: []string{"name", "until"},
			Sample:       map[string]string{"name": "香港8区cn2-无防专线", "until": "2026-10-26"},
		},
		{
			ID: "load", Title: "⚠️ 负载告警",
			Body: "{{rule}}\n\n{{metric}}  达到  {{threshold}}\n达标比例  {{ratio}}\n\n节点\n{{nodes}}",
			Node: "{{name}}", NodeJoin: "\n",
			Placeholders:     []string{"rule", "metric", "threshold", "ratio", "nodes"},
			NodePlaceholders: []string{"name"},
			Sample: map[string]string{
				"rule": "CPU 持续偏高", "metric": "CPU", "threshold": "80", "ratio": "80%",
			},
			SampleNodes: []map[string]string{{"name": "netcup RS-不赖ai"}, {"name": "日本东京精品"}},
		},
		{
			ID: "login", Title: "🔑 登录提醒",
			Body:         "方式\n{{method}}\n\n来源\n{{ip}}\n{{location}}\n\n设备\n{{device}}",
			Placeholders: []string{"method", "ip", "location", "device"},
			Sample: map[string]string{
				"method": "password", "ip": "203.0.113.10", "location": "香港",
				"device": "Mozilla/5.0",
			},
		},
		{
			ID: "test", Title: "✅ 测试消息",
			Body: "这是一条测试消息。\n\n如果你能看到标题、空行和时间，说明通知格式已经生效。",
		},
	}
}

func report(id, title string) Definition {
	return Definition{
		ID: id, Title: title,
		Body:             "📅 {{period}}\n\n{{nodes}}\n\n合计  {{count}} 台  ·  {{total}}",
		Node:             "{{name}}\n{{used}}",
		NodeJoin:         "\n\n",
		Placeholders:     []string{"period", "nodes", "count", "total"},
		NodePlaceholders: []string{"name", "used"},
		Sample:           map[string]string{"period": "2026-09-25", "count": "2", "total": "16.68 GB"},
		SampleNodes: []map[string]string{
			{"name": "OVHCloud 塔菲API源站机 美西", "used": "14.82 GB"},
			{"name": "netcup RS-x社区+各种服务基建", "used": "1.86 GB"},
		},
	}
}

func Find(id string) (Definition, bool) {
	for _, item := range Catalog() {
		if item.ID == id {
			return item, true
		}
	}
	return Definition{}, false
}

// Render 用已保存的模板渲染一条通知。没有保存过时使用内置默认模板。
func Render(id string, data map[string]string, nodes []map[string]string) (title, body string, err error) {
	def, ok := Find(id)
	if !ok {
		return "", "", fmt.Errorf("unknown notification template %q", id)
	}
	stored, customized := override(id)
	title, body = render(def, stored, customized, data, nodes)
	return title, body, nil
}

// List 返回全部模板和按样例数据生成的预览。
func List() ([]View, error) {
	saved := load()
	views := make([]View, 0, len(Catalog()))
	for _, def := range Catalog() {
		views = append(views, viewFor(def, saved[def.ID], hasOverride(saved, def.ID)))
	}
	return views, nil
}

// Save 保存一种通知的模板。
func Save(id, title, body, node string) error {
	def, ok := Find(id)
	if !ok {
		return fmt.Errorf("unknown notification template %q", id)
	}
	if len([]rune(title)) > maxTitleLen || len([]rune(body)) > maxBodyLen || len([]rune(node)) > maxNodeLen {
		return fmt.Errorf("template is too long")
	}
	if !def.hasNode() {
		node = ""
	}
	saved := load()
	saved[id] = Stored{Title: title, Body: body, Node: node}
	return config.Set(config.NotificationEventTemplatesKey, saved)
}

// Reset 恢复一种通知的内置模板。
func Reset(id string) error {
	if _, ok := Find(id); !ok {
		return fmt.Errorf("unknown notification template %q", id)
	}
	saved := load()
	delete(saved, id)
	return config.Set(config.NotificationEventTemplatesKey, saved)
}

// PreviewDraft 用编辑中的文本和样例数据渲染，不写入数据库。
func PreviewDraft(id, title, body, node string) (Preview, error) {
	def, ok := Find(id)
	if !ok {
		return Preview{}, fmt.Errorf("unknown notification template %q", id)
	}
	if !def.hasNode() {
		node = ""
	}
	renderedTitle, renderedBody := render(def, Stored{Title: title, Body: body, Node: node}, true, def.Sample, def.SampleNodes)
	return composePreview(renderedTitle, renderedBody), nil
}

func viewFor(def Definition, stored Stored, customized bool) View {
	title, body, node := def.Title, def.Body, def.Node
	if customized {
		title, body, node = stored.Title, stored.Body, stored.Node
	}
	renderedTitle, renderedBody := render(def, Stored{Title: title, Body: body, Node: node}, true, def.Sample, def.SampleNodes)
	return View{
		ID: def.ID, Title: title, Body: body, Node: node,
		HasNode: def.hasNode(), Customized: customized,
		Placeholders: def.Placeholders, NodePlaceholders: def.NodePlaceholders,
		Sample: def.Sample, SampleNodes: def.SampleNodes,
		Preview: composePreview(renderedTitle, renderedBody),
	}
}

func render(def Definition, stored Stored, customized bool, data map[string]string, nodes []map[string]string) (string, string) {
	title, body, node := def.Title, def.Body, def.Node
	if customized {
		title, body, node = stored.Title, stored.Body, stored.Node
	}
	if data == nil {
		data = map[string]string{}
	}
	join := def.NodeJoin
	if join == "" {
		join = "\n\n"
	}
	renderedNodes := make([]string, 0, len(nodes))
	for _, item := range nodes {
		renderedNodes = append(renderedNodes, apply(node, item))
	}
	data["nodes"] = strings.Join(renderedNodes, join)
	return apply(title, data), apply(body, data)
}

func apply(template string, data map[string]string) string {
	if template == "" || len(data) == 0 {
		return template
	}
	replacer := make([]string, 0, len(data)*2)
	for key, value := range data {
		replacer = append(replacer, "{{"+key+"}}", value)
	}
	return strings.NewReplacer(replacer...).Replace(template)
}

func composePreview(title, body string) Preview {
	loc := notifytime.Location()
	text, err := messageSender.Compose(models.EventMessage{
		Title:   title,
		Message: body,
		Time:    time.Date(2026, 9, 26, 0, 0, 0, 0, loc),
	})
	if err != nil || strings.TrimSpace(text) == "" {
		text = body
	}
	return Preview{Title: title, Text: strings.TrimRight(text, "\n")}
}

func (d Definition) hasNode() bool {
	return d.Node != "" || len(d.NodePlaceholders) > 0
}

func override(id string) (Stored, bool) {
	saved := load()
	item, ok := saved[id]
	return item, ok
}

func hasOverride(saved map[string]Stored, id string) bool {
	_, ok := saved[id]
	return ok
}

func load() map[string]Stored {
	if !config.Available() {
		return map[string]Stored{}
	}
	saved, err := config.GetAs[map[string]Stored](config.NotificationEventTemplatesKey, map[string]Stored{})
	if err != nil || saved == nil {
		return map[string]Stored{}
	}
	return saved
}
