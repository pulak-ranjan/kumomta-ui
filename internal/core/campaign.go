package core

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net"
	"net/smtp"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// CampaignService handles bulk sending logic
type CampaignService struct {
	Store *store.Store
}

func NewCampaignService(st *store.Store) *CampaignService {
	return &CampaignService{Store: st}
}

// ImportRecipientsFromCSV parses a CSV and adds recipients to a campaign
func (cs *CampaignService) ImportRecipientsFromCSV(campaignID uint, r io.Reader) error {
	reader := csv.NewReader(r)

	var recipients []models.CampaignRecipient
	batchSize := 500

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(record) < 1 { continue }
		email := strings.TrimSpace(record[0])

		// Basic validation
		if email == "" || !strings.Contains(email, "@") { continue }

		recipients = append(recipients, models.CampaignRecipient{
			CampaignID: campaignID,
			Email:      email,
			Status:     "pending",
		})

		if len(recipients) >= batchSize {
			if err := cs.Store.DB.Create(&recipients).Error; err != nil {
				log.Printf("Failed to batch import: %v", err)
			}
			recipients = nil // Reset slice
		}
	}

	// Insert remaining
	if len(recipients) > 0 {
		if err := cs.Store.DB.Create(&recipients).Error; err != nil {
			log.Printf("Failed to batch import remainder: %v", err)
		}
	}
	return nil
}

// StartCampaign launches the sending process in a background goroutine
func (cs *CampaignService) StartCampaign(campaignID uint) error {
	var campaign models.Campaign
	if err := cs.Store.DB.Preload("Sender").Preload("Sender.Domain").First(&campaign, campaignID).Error; err != nil {
		return err
	}

	if campaign.Status == "sending" || campaign.Status == "completed" {
		return fmt.Errorf("campaign is already %s", campaign.Status)
	}

	// Update status
	campaign.Status = "sending"
	cs.Store.DB.Save(&campaign)

	go cs.processCampaign(campaign)

	return nil
}

// ResumeInterruptedCampaigns finds campaigns stuck in "sending" and restarts them
func (cs *CampaignService) ResumeInterruptedCampaigns() error {
	var campaigns []models.Campaign
	if err := cs.Store.DB.Where("status = 'sending'").Find(&campaigns).Error; err != nil {
		return err
	}

	for _, c := range campaigns {
		// Re-load sender details
		if err := cs.Store.DB.Preload("Sender").Preload("Sender.Domain").First(&c, c.ID).Error; err != nil {
			log.Printf("Failed to reload campaign %d: %v", c.ID, err)
			continue
		}
		log.Printf("Resuming campaign %d: %s", c.ID, c.Name)
		go cs.processCampaign(c)
	}
	return nil
}

// StartScheduledCampaigns finds campaigns ready to send and launches them
func (cs *CampaignService) StartScheduledCampaigns() error {
	var campaigns []models.Campaign
	now := time.Now()

	if err := cs.Store.DB.Where("status = 'scheduled' AND scheduled_at <= ?", now).Find(&campaigns).Error; err != nil {
		return err
	}

	for _, c := range campaigns {
		// Atomic update to prevent double-send race conditions
		result := cs.Store.DB.Model(&c).Where("status = 'scheduled'").Update("status", "sending")
		if result.RowsAffected == 0 {
			continue // Already picked up by another worker?
		}

		// Reload full object with preload
		if err := cs.Store.DB.Preload("Sender").Preload("Sender.Domain").First(&c, c.ID).Error; err != nil {
			log.Printf("Failed to load scheduled campaign %d: %v", c.ID, err)
			continue
		}

		log.Printf("Starting scheduled campaign %d: %s", c.ID, c.Name)
		go cs.processCampaign(c)
	}
	return nil
}

func (cs *CampaignService) processCampaign(c models.Campaign) {
	batchSize := 100
	sender := c.Sender
	addr := "127.0.0.1:25"

	// Construct message common headers
	safeSubject := strings.ReplaceAll(c.Subject, "\r", "")
	safeSubject = strings.ReplaceAll(safeSubject, "\n", "")
	baseHeaders := fmt.Sprintf("From: %s\r\nSubject: %s\r\nX-Campaign: %d\r\nX-Kumo-Ref: Bulk\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		sender.Email, safeSubject, c.ID)

	// Determine Base URL for tracking
	baseURL := "http://localhost:9000"
	if settings, err := cs.Store.GetSettings(); err == nil && settings.MainHostname != "" {
		protocol := "https"
		if settings.MainHostname == "localhost" { protocol = "http" }
		baseURL = fmt.Sprintf("%s://%s", protocol, settings.MainHostname)
	}

	// Persistent Connection
	// We use DialTimeout to avoid hanging if local MTA is stuck
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		log.Printf("Campaign %d: Failed to connect to SMTP: %v", c.ID, err)
		c.Status = "failed"
		cs.Store.DB.Save(&c)
		return
	}

	client, err := smtp.NewClient(conn, "localhost")
	if err != nil {
		conn.Close()
		log.Printf("Campaign %d: SMTP Client handshake failed: %v", c.ID, err)
		c.Status = "failed"
		cs.Store.DB.Save(&c)
		return
	}
	defer client.Quit()

	for {
		var recipients []models.CampaignRecipient
		if err := cs.Store.DB.Where("campaign_id = ? AND status = 'pending'", c.ID).Limit(batchSize).Find(&recipients).Error; err != nil {
			log.Printf("DB Error fetching recipients: %v", err)
			break
		}

		if len(recipients) == 0 {
			// No more pending recipients -> Completed
			c.Status = "completed"
			cs.Store.DB.Save(&c)
			return
		}

		for _, r := range recipients {
			// Reconnection Logic
			if err := client.Mail(sender.Email); err != nil {
				log.Printf("SMTP Connection lost (%v). Reconnecting...", err)
				client.Close()

				conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
				if err == nil {
					client, err = smtp.NewClient(conn, "localhost")
				}

				if err != nil {
					log.Printf("Failed to reconnect: %v. Pausing campaign.", err)
					c.Status = "paused" // Mark paused to be resumed manually or by scheduler
					cs.Store.DB.Save(&c)
					return
				}
				// Retry MAIL command
				if err := client.Mail(sender.Email); err != nil {
					log.Printf("Reconnection failed: %v", err)
					c.Status = "paused"
					cs.Store.DB.Save(&c)
					return
				}
			}

			if err := client.Rcpt(r.Email); err != nil {
				// Recipient rejected (e.g. invalid syntax, or server block)
				// We fail this specific recipient but continue
				r.Status = "failed"
				r.Error = err.Error()
				cs.Store.DB.Save(&r)
				client.Reset()
				continue
			}

			wc, err := client.Data()
			if err != nil {
				log.Printf("SMTP Data error: %v", err)
				client.Reset()
				continue
			}

			// Inject Tracking Pixel & Rewrite Links
			trackingOpenURL := fmt.Sprintf("%s/api/track/open/%d", baseURL, r.ID)
			pixel := fmt.Sprintf(`<img src="%s" alt="" width="1" height="1" style="display:none" />`, trackingOpenURL)

			// Rewrite Links for Click Tracking
			bodyWithLinks := rewriteLinks(c.Body, baseURL, r.ID)

			bodyFinal := bodyWithLinks + "\n" + pixel

			msg := fmt.Sprintf("To: %s\r\n%s%s", r.Email, baseHeaders, bodyFinal)

			if _, err = wc.Write([]byte(msg)); err != nil {
				log.Printf("SMTP Write error: %v", err)
			}
			if err = wc.Close(); err != nil {
				log.Printf("SMTP Close error: %v", err)
			}

			// Success
			r.Status = "sent"
			r.SentAt = time.Now()
			cs.Store.DB.Save(&r)

			c.TotalSent++
			// Throttle
			time.Sleep(10 * time.Millisecond)
		}

		// Update stats after batch
		cs.Store.DB.Model(&c).Updates(map[string]interface{}{
			"total_sent": c.TotalSent,
			"total_failed": c.TotalFailed,
		})
	}
}

// rewriteLinks finds all href="..." and replaces them with tracking URLs
func rewriteLinks(html string, baseURL string, recipientID uint) string {
	// Simple regex for href attributes
	// Note: This is a basic implementation. A proper HTML parser would be more robust but heavier.
	re := regexp.MustCompile(`href=["'](http[^"']+)["']`)

	return re.ReplaceAllStringFunc(html, func(match string) string {
		// Extract URL from match (e.g. href="http://google.com")
		// We need to be careful with quotes
		quote := match[5:6] // " or '
		originalURL := match[6 : len(match)-1]

		// Encode the original URL
		encodedURL := url.QueryEscape(originalURL)

		// Construct tracking URL
		// We sign the original URL to prevent tampering/open redirects
		signature := SignLink(originalURL)
		trackingURL := fmt.Sprintf("%s/api/track/click/%d?url=%s&sig=%s", baseURL, recipientID, encodedURL, signature)

		return fmt.Sprintf("href=%s%s%s", quote, trackingURL, quote)
	})
}

// SendSingleEmail sends a transactional email via local KumoMTA
func (cs *CampaignService) SendSingleEmail(to string, subject string, body string, senderID uint) error {
	var sender models.Sender
	if err := cs.Store.DB.Preload("Domain").First(&sender, senderID).Error; err != nil {
		return fmt.Errorf("sender not found: %v", err)
	}

	addr := "127.0.0.1:25"
	// Use DialTimeout for robustness
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, "localhost")
	if err != nil {
		conn.Close()
		return err
	}
	defer client.Quit()

	if err := client.Mail(sender.Email); err != nil { return err }
	if err := client.Rcpt(to); err != nil { return err }

	wc, err := client.Data()
	if err != nil { return err }

	// Simple Headers
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		sender.Email, to, subject)

	msg := []byte(headers + body)

	if _, err = wc.Write(msg); err != nil { return err }
	return wc.Close()
}
