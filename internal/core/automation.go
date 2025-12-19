package core

import (
	"encoding/json"
	"log"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type AutomationService struct {
	Store *store.Store
}

func NewAutomationService(st *store.Store) *AutomationService {
	return &AutomationService{Store: st}
}

// TriggerWorkflow finds workflows matching the trigger and executes them
func (as *AutomationService) TriggerWorkflow(trigger string, contactID uint) {
	var workflows []models.AutomationWorkflow
	if err := as.Store.DB.Where("trigger = ? AND is_active = ?", trigger, true).Find(&workflows).Error; err != nil {
		return
	}

	for _, wf := range workflows {
		go as.runWorkflow(wf, contactID)
	}
}

func (as *AutomationService) runWorkflow(wf models.AutomationWorkflow, contactID uint) {
	// 1. Parse Steps
	var steps []map[string]interface{}
	if err := json.Unmarshal([]byte(wf.StepsJSON), &steps); err != nil {
		log.Printf("Automation %d: Invalid steps JSON", wf.ID)
		return
	}

	// 2. Execute Steps Sequentially
	for _, step := range steps {
		stepType, _ := step["type"].(string)

		switch stepType {
		case "wait":
			// simple wait logic
			if seconds, ok := step["seconds"].(float64); ok {
				time.Sleep(time.Duration(seconds) * time.Second)
			}
		case "send_email":
			subject, _ := step["subject"].(string)
			body, _ := step["body"].(string)
			senderIDFloat, _ := step["sender_id"].(float64)
			senderID := uint(senderIDFloat)

			// Resolve contact email
			var contact models.Contact
			if err := as.Store.DB.First(&contact, contactID).Error; err == nil {
				cs := NewCampaignService(as.Store)
				if err := cs.SendSingleEmail(contact.Email, subject, body, senderID); err != nil {
					log.Printf("Auto: Failed to send email to %s: %v", contact.Email, err)
				} else {
					log.Printf("Auto: Sent email to %s", contact.Email)
				}
			}
		case "send_whatsapp":
			log.Printf("Auto: Sending WhatsApp to contact %d", contactID)
		}
	}
}
