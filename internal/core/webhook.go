package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// WebhookService handles sending notifications
type WebhookService struct {
	Store *store.Store
}

func NewWebhookService(st *store.Store) *WebhookService {
	return &WebhookService{Store: st}
}

// SlackMessage represents a Slack webhook payload
type SlackMessage struct {
	Text        string       `json:"text,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color  string  `json:"color"`
	Title  string  `json:"title"`
	Text   string  `json:"text"`
	Fields []Field `json:"fields,omitempty"`
	Footer string  `json:"footer,omitempty"`
	Ts     int64   `json:"ts,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// DiscordMessage represents a Discord webhook payload
type DiscordMessage struct {
	Content string         `json:"content,omitempty"`
	Username string        `json:"username,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Color       int           `json:"color"`
	Fields      []DiscordField `json:"fields,omitempty"`
	Footer      *DiscordFooter `json:"footer,omitempty"`
	Timestamp   string        `json:"timestamp,omitempty"`
}

type DiscordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordFooter struct {
	Text string `json:"text"`
}

// SendBounceAlert sends an alert when bounce rate is high
func (ws *WebhookService) SendBounceAlert(domain string, bounceRate float64, sent, bounced int64) error {
	settings, err := ws.Store.GetSettings()
	if err != nil || settings == nil || !settings.WebhookEnabled || settings.WebhookURL == "" {
		return nil
	}

	// Check if bounce rate exceeds threshold
	if bounceRate < settings.BounceAlertPct {
		return nil
	}

	isDiscord := strings.Contains(settings.WebhookURL, "discord.com")

	var payload []byte
	var eventType = "bounce_alert"

	if isDiscord {
		msg := DiscordMessage{
			Username: "KumoMTA Alert",
			Embeds: []DiscordEmbed{
				{
					Title:       "⚠️ High Bounce Rate Alert",
					Description: fmt.Sprintf("Domain **%s** has a high bounce rate", domain),
					Color:       15158332, // Red
					Fields: []DiscordField{
						{Name: "Bounce Rate", Value: fmt.Sprintf("%.2f%%", bounceRate), Inline: true},
						{Name: "Total Sent", Value: fmt.Sprintf("%d", sent), Inline: true},
						{Name: "Bounced", Value: fmt.Sprintf("%d", bounced), Inline: true},
					},
					Footer:    &DiscordFooter{Text: "KumoMTA UI"},
					Timestamp: time.Now().Format(time.RFC3339),
				},
			},
		}
		payload, _ = json.Marshal(msg)
	} else {
		// Slack format
		msg := SlackMessage{
			Username:  "KumoMTA Alert",
			IconEmoji: ":warning:",
			Attachments: []Attachment{
				{
					Color: "danger",
					Title: "⚠️ High Bounce Rate Alert",
					Text:  fmt.Sprintf("Domain *%s* has a high bounce rate", domain),
					Fields: []Field{
						{Title: "Bounce Rate", Value: fmt.Sprintf("%.2f%%", bounceRate), Short: true},
						{Title: "Total Sent", Value: fmt.Sprintf("%d", sent), Short: true},
						{Title: "Bounced", Value: fmt.Sprintf("%d", bounced), Short: true},
					},
					Footer: "KumoMTA UI",
					Ts:     time.Now().Unix(),
				},
			},
		}
		payload, _ = json.Marshal(msg)
	}

	return ws.send(settings.WebhookURL, payload, eventType)
}

// SendDailySummary sends a daily stats summary
func (ws *WebhookService) SendDailySummary(stats map[string]DayStats) error {
	settings, err := ws.Store.GetSettings()
	if err != nil || settings == nil || !settings.WebhookEnabled || settings.WebhookURL == "" {
		return nil
	}

	isDiscord := strings.Contains(settings.WebhookURL, "discord.com")

	totalSent := int64(0)
	totalDelivered := int64(0)
	totalBounced := int64(0)

	for _, s := range stats {
		totalSent += s.Sent
		totalDelivered += s.Delivered
		totalBounced += s.Bounced
	}

	deliveryRate := float64(0)
	if totalSent > 0 {
		deliveryRate = float64(totalDelivered) / float64(totalSent) * 100
	}

	var payload []byte
	eventType := "daily_summary"

	if isDiscord {
		msg := DiscordMessage{
			Username: "KumoMTA Report",
			Embeds: []DiscordEmbed{
				{
					Title:       "📊 Daily Sending Summary",
					Description: fmt.Sprintf("Stats for %s", time.Now().Format("2006-01-02")),
					Color:       3447003, // Blue
					Fields: []DiscordField{
						{Name: "Total Sent", Value: fmt.Sprintf("%d", totalSent), Inline: true},
						{Name: "Delivered", Value: fmt.Sprintf("%d", totalDelivered), Inline: true},
						{Name: "Bounced", Value: fmt.Sprintf("%d", totalBounced), Inline: true},
						{Name: "Delivery Rate", Value: fmt.Sprintf("%.2f%%", deliveryRate), Inline: true},
						{Name: "Domains Active", Value: fmt.Sprintf("%d", len(stats)), Inline: true},
					},
					Footer:    &DiscordFooter{Text: "KumoMTA UI"},
					Timestamp: time.Now().Format(time.RFC3339),
				},
			},
		}
		payload, _ = json.Marshal(msg)
	} else {
		msg := SlackMessage{
			Username:  "KumoMTA Report",
			IconEmoji: ":bar_chart:",
			Attachments: []Attachment{
				{
					Color: "good",
					Title: "📊 Daily Sending Summary",
					Text:  fmt.Sprintf("Stats for %s", time.Now().Format("2006-01-02")),
					Fields: []Field{
						{Title: "Total Sent", Value: fmt.Sprintf("%d", totalSent), Short: true},
						{Title: "Delivered", Value: fmt.Sprintf("%d", totalDelivered), Short: true},
						{Title: "Bounced", Value: fmt.Sprintf("%d", totalBounced), Short: true},
						{Title: "Delivery Rate", Value: fmt.Sprintf("%.2f%%", deliveryRate), Short: true},
					},
					Footer: "KumoMTA UI",
					Ts:     time.Now().Unix(),
				},
			},
		}
		payload, _ = json.Marshal(msg)
	}

	return ws.send(settings.WebhookURL, payload, eventType)
}

// SendTestWebhook sends a test message
func (ws *WebhookService) SendTestWebhook(webhookURL string) error {
	isDiscord := strings.Contains(webhookURL, "discord.com")

	var payload []byte

	if isDiscord {
		msg := DiscordMessage{
			Username: "KumoMTA Test",
			Embeds: []DiscordEmbed{
				{
					Title:       "✅ Webhook Test Successful",
					Description: "Your KumoMTA UI webhook is configured correctly!",
					Color:       5763719, // Green
					Footer:      &DiscordFooter{Text: "KumoMTA UI"},
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		}
		payload, _ = json.Marshal(msg)
	} else {
		msg := SlackMessage{
			Username:  "KumoMTA Test",
			IconEmoji: ":white_check_mark:",
			Text:      "✅ Webhook Test Successful! Your KumoMTA UI webhook is configured correctly.",
		}
		payload, _ = json.Marshal(msg)
	}

	return ws.send(webhookURL, payload, "test")
}

func (ws *WebhookService) send(url string, payload []byte, eventType string) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ws.logWebhook(eventType, string(payload), 0, err.Error())
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	ws.logWebhook(eventType, string(payload), resp.StatusCode, string(body))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (ws *WebhookService) logWebhook(eventType, payload string, status int, response string) {
	log := &models.WebhookLog{
		EventType: eventType,
		Payload:   payload,
		Status:    status,
		Response:  response,
		CreatedAt: time.Now(),
	}
	ws.Store.CreateWebhookLog(log)
}

// CheckBounceRates checks all domains and sends alerts if needed
func (ws *WebhookService) CheckBounceRates() error {
	stats, err := GetAllDomainsStats(1) // Today only
	if err != nil {
		return err
	}

	for domain, dayStats := range stats {
		if len(dayStats) == 0 {
			continue
		}

		today := dayStats[len(dayStats)-1]
		if today.Sent == 0 {
			continue
		}

		bounceRate := float64(today.Bounced) / float64(today.Sent) * 100
		ws.SendBounceAlert(domain, bounceRate, today.Sent, today.Bounced)
	}

	return nil
}
