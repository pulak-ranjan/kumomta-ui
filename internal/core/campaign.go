package core

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/smtp"
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

func (cs *CampaignService) processCampaign(c models.Campaign) {
	var recipients []models.CampaignRecipient
	// Fetch pending recipients
	cs.Store.DB.Where("campaign_id = ? AND status = 'pending'", c.ID).Find(&recipients)

	sender := c.Sender
	addr := "127.0.0.1:25"

	// Open persistent SMTP connection
	client, err := smtp.Dial(addr)
	if err != nil {
		log.Printf("Campaign %d: Failed to connect to SMTP: %v", c.ID, err)
		return
	}
	defer client.Quit()

	// Construct message common headers
	// Security: Sanitize subject to prevent header injection
	safeSubject := strings.ReplaceAll(c.Subject, "\r", "")
	safeSubject = strings.ReplaceAll(safeSubject, "\n", "")

	baseHeaders := fmt.Sprintf("From: %s\r\nSubject: %s\r\nX-Campaign: %d\r\nX-Kumo-Ref: Bulk\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n",
		sender.Email, safeSubject, c.ID)

	for _, r := range recipients {
		// Send via persistent SMTP connection
		if err := client.Mail(sender.Email); err != nil {
			log.Printf("SMTP Mail error: %v", err)
			break
		}
		if err := client.Rcpt(r.Email); err != nil {
			log.Printf("SMTP Rcpt error: %v", err)
			client.Reset()
			continue
		}
		wc, err := client.Data()
		if err != nil {
			log.Printf("SMTP Data error: %v", err)
			break
		}

		// Inject Tracking Pixel
		baseURL := "http://localhost:9000"
		if settings, err := cs.Store.GetSettings(); err == nil && settings.MainHostname != "" {
			// Assuming MainHostname is just the domain (e.g. mta.example.com).
			// We need schema. Usually if certs are installed it's https.
			// Ideally we store a full BaseURL or derive it. For now, assume HTTPS for safety if not localhost.
			protocol := "https"
			if settings.MainHostname == "localhost" { protocol = "http" }
			baseURL = fmt.Sprintf("%s://%s", protocol, settings.MainHostname)
		}

		trackingURL := fmt.Sprintf("%s/api/track/open/%d", baseURL, r.ID)
		pixel := fmt.Sprintf(`<img src="%s" alt="" width="1" height="1" style="display:none" />`, trackingURL)

		bodyWithPixel := c.Body + "\n" + pixel

		// Body
		msg := fmt.Sprintf("To: %s\r\n%s%s", r.Email, baseHeaders, bodyWithPixel)

		if _, err = wc.Write([]byte(msg)); err != nil {
			log.Printf("SMTP Write error: %v", err)
		}
		if err = wc.Close(); err != nil {
			log.Printf("SMTP Close error: %v", err)
		}

		r.SentAt = time.Now()
		// Simple error check on the last operation, mostly
		if err != nil {
			r.Status = "failed"
			r.Error = err.Error()
			c.TotalFailed++
		} else {
			r.Status = "sent"
			r.Error = ""
			c.TotalSent++
		}

		// Save recipient status
		cs.Store.DB.Save(&r)

		// Update Campaign stats
		cs.Store.DB.Model(&c).Updates(map[string]interface{}{
			"total_sent": c.TotalSent,
			"total_failed": c.TotalFailed,
		})

		// Throttle slightly
		time.Sleep(10 * time.Millisecond)
	}

	c.Status = "completed"
	cs.Store.DB.Save(&c)
}
