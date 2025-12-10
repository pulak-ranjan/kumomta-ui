package core

import (
	"fmt"
	"log"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

// Schedules mapping: Day Number (1-based) -> KumoMTA Rate String
// Day 1 starts low. If a sender is on Day 5, they get that rate.
var WarmupSchedules = map[string][]string{
	"conservative": {
		"10/hr", "20/hr", "40/hr", "80/hr", "150/hr", "300/hr", "600/hr", "1000/hr", "2000/hr", "4000/hr",
	},
	"standard": {
		"25/hr", "50/hr", "100/hr", "200/hr", "400/hr", "800/hr", "1600/hr", "3200/hr", "6400/hr", "12000/hr",
	},
	"aggressive": {
		"50/hr", "100/hr", "250/hr", "500/hr", "1000/hr", "2500/hr", "5000/hr", "10000/hr", "20000/hr",
	},
}

// ProcessDailyWarmup checks all senders and advances their schedule if 24h passed
func ProcessDailyWarmup(st *store.Store) error {
	domains, err := st.ListDomains()
	if err != nil {
		return err
	}

	updatesMade := false

	for _, d := range domains {
		for _, s := range d.Senders {
			if !s.WarmupEnabled {
				continue
			}

			// Initialize timestamp if missing (first run)
			if s.WarmupLastUpdate.IsZero() {
				s.WarmupLastUpdate = time.Now()
				// Save initial state without advancing day
				if err := st.UpdateSender(&s); err != nil {
					log.Printf("[Warmup] DB Error init %s: %v", s.Email, err)
				}
				continue
			}

			// Check if 24 hours have passed since last increment
			if time.Since(s.WarmupLastUpdate) < 24*time.Hour {
				continue
			}

			// Get the plan (default to standard if missing)
			planName := s.WarmupPlan
			if planName == "" { planName = "standard" }
			
			plan, exists := WarmupSchedules[planName]
			if !exists { plan = WarmupSchedules["standard"] }

			// Check if we should Advance or Finish
			if s.WarmupDay >= len(plan) {
				log.Printf("[Warmup] Sender %s completed %s plan. Disabling warmup limits.", s.Email, planName)
				s.WarmupEnabled = false // Done! They run unlimited now.
			} else {
				s.WarmupDay++
				s.WarmupLastUpdate = time.Now()
				log.Printf("[Warmup] Bumped %s to Day %d (%s)", s.Email, s.WarmupDay, plan[s.WarmupDay-1])
			}

			// Save progress
			if err := st.UpdateSender(&s); err != nil {
				log.Printf("[Warmup] DB Error saving %s: %v", s.Email, err)
			}
			updatesMade = true
		}
	}

	// If any rates changed, we MUST regenerate Kumo config to apply them
	if updatesMade {
		log.Println("[Warmup] Applying new rate limits to KumoMTA...")
		snap, err := LoadSnapshot(st)
		if err == nil {
			if _, err := ApplyKumoConfig(snap); err != nil {
				return fmt.Errorf("failed to apply warmup config: %v", err)
			}
		}
	}

	return nil
}

// GetSenderRate returns the current KumoMTA rate string (e.g. "100/hr") for a sender
// This is called by configgen.go
func GetSenderRate(s models.Sender) string {
	if !s.WarmupEnabled {
		return "" // Empty means no limit
	}
	
	planName := s.WarmupPlan
	if planName == "" { planName = "standard" }

	plan, ok := WarmupSchedules[planName]
	if !ok { return "" }

	// Calculate array index (Day 1 = index 0)
	dayIndex := s.WarmupDay - 1
	
	// Safety bounds
	if dayIndex < 0 { dayIndex = 0 }
	if dayIndex >= len(plan) { return "" } // Should have been disabled, but safe fallback

	return plan[dayIndex]
}
