package core

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pulak-ranjan/kumomta-ui/internal/models"
)

const (
	KumoSpoolData = "/var/spool/kumomta/data"
	KumoSpoolMeta = "/var/spool/kumomta/meta"
)

// QueueMetaEntry represents metadata stored in spool
type QueueMetaEntry struct {
	ID        string                 `json:"id"`
	Sender    string                 `json:"sender"`
	Recipient string                 `json:"recipient"`
	Meta      map[string]interface{} `json:"meta"`
	Created   time.Time              `json:"created"`
	Due       time.Time              `json:"due"`
	Attempts  int                    `json:"num_attempts"`
	LastError string                 `json:"last_error"`
}

// GetQueueMessages reads the KumoMTA spool and returns queue entries
func GetQueueMessages(limit int) ([]models.QueueMessage, error) {
	var messages []models.QueueMessage

	// Try to read from meta spool first
	err := filepath.WalkDir(KumoSpoolMeta, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".meta") && !strings.HasSuffix(path, ".json") {
			return nil
		}
		if len(messages) >= limit {
			return filepath.SkipAll
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var entry QueueMetaEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil
		}

		// Get file info for size
		info, _ := d.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}

		msg := models.QueueMessage{
			ID:          entry.ID,
			Sender:      entry.Sender,
			Recipient:   entry.Recipient,
			Size:        size,
			CreatedAt:   entry.Created,
			Attempts:    entry.Attempts,
			NextRetry:   entry.Due,
			ErrorMsg:    entry.LastError,
			Status:      "queued",
		}

		if entry.Attempts > 0 {
			msg.Status = "deferred"
		}

		// Try to extract subject from meta
		if entry.Meta != nil {
			if subj, ok := entry.Meta["Subject"].(string); ok {
				msg.Subject = subj
			}
		}

		messages = append(messages, msg)
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		// If meta spool doesn't exist, try alternative method
	}

	// Alternative: scan data spool for file counts
	if len(messages) == 0 {
		messages = scanDataSpool(limit)
	}

	// Sort by created time (newest first)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].CreatedAt.After(messages[j].CreatedAt)
	})

	return messages, nil
}

func scanDataSpool(limit int) []models.QueueMessage {
	var messages []models.QueueMessage

	filepath.WalkDir(KumoSpoolData, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if len(messages) >= limit {
			return filepath.SkipAll
		}

		info, _ := d.Info()
		if info == nil {
			return nil
		}

		msg := models.QueueMessage{
			ID:        filepath.Base(path),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
			Status:    "queued",
		}
		messages = append(messages, msg)
		return nil
	})

	return messages
}

// GetQueueStats returns summary of queue
type QueueStats struct {
	Total    int   `json:"total"`
	Queued   int   `json:"queued"`
	Deferred int   `json:"deferred"`
	TotalSize int64 `json:"total_size"`
	OldestMsg time.Time `json:"oldest_msg"`
}

func GetQueueStats() (*QueueStats, error) {
	stats := &QueueStats{}

	filepath.WalkDir(KumoSpoolMeta, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		stats.Total++

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var entry QueueMetaEntry
		if json.Unmarshal(data, &entry) == nil {
			if entry.Attempts > 0 {
				stats.Deferred++
			} else {
				stats.Queued++
			}
			if stats.OldestMsg.IsZero() || entry.Created.Before(stats.OldestMsg) {
				stats.OldestMsg = entry.Created
			}
		}

		if info, _ := d.Info(); info != nil {
			stats.TotalSize += info.Size()
		}

		return nil
	})

	// Also count data spool
	filepath.WalkDir(KumoSpoolData, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, _ := d.Info(); info != nil {
			stats.TotalSize += info.Size()
		}
		return nil
	})

	return stats, nil
}

// DeleteQueueMessage removes a message from the queue
func DeleteQueueMessage(id string) error {
	// Delete from meta spool
	metaPath := filepath.Join(KumoSpoolMeta, id)
	os.Remove(metaPath)
	os.Remove(metaPath + ".meta")
	os.Remove(metaPath + ".json")

	// Delete from data spool
	dataPath := filepath.Join(KumoSpoolData, id)
	os.Remove(dataPath)

	return nil
}

// FlushQueue attempts to retry all deferred messages
func FlushQueue() error {
	// KumoMTA doesn't have a direct flush command,
	// but we can restart the service or use the HTTP API
	// For now, this is a placeholder
	return nil
}
