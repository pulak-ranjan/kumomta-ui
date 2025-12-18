package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumomta-ui/internal/core"
	"github.com/pulak-ranjan/kumomta-ui/internal/models"
	"github.com/pulak-ranjan/kumomta-ui/internal/store"
)

type CampaignHandler struct {
	Store   *store.Store
	Service *core.CampaignService
}

func NewCampaignHandler(st *store.Store) *CampaignHandler {
	return &CampaignHandler{
		Store:   st,
		Service: core.NewCampaignService(st),
	}
}

// Routes registers the campaign API routes
func (h *CampaignHandler) Routes(r chi.Router) {
	r.Get("/", h.listCampaigns)
	r.Post("/", h.createCampaign)
	r.Post("/{id}/import", h.importRecipients)
	r.Post("/{id}/send", h.startCampaign)
	r.Get("/{id}", h.getCampaign)
}

func (h *CampaignHandler) listCampaigns(w http.ResponseWriter, r *http.Request) {
	var campaigns []models.Campaign
	// Order by newest first
	if err := h.Store.DB.Order("created_at desc").Find(&campaigns).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	writeJSON(w, http.StatusOK, campaigns)
}

func (h *CampaignHandler) createCampaign(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Subject  string `json:"subject"`
		Body     string `json:"body"`
		SenderID uint   `json:"sender_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	campaign := models.Campaign{
		Name:     req.Name,
		Subject:  req.Subject,
		Body:     req.Body,
		SenderID: req.SenderID,
		Status:   "draft",
	}

	if err := h.Store.DB.Create(&campaign).Error; err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create campaign"})
		return
	}

	writeJSON(w, http.StatusCreated, campaign)
}

func (h *CampaignHandler) importRecipients(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	// Max 10MB CSV
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too big"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file"})
		return
	}
	defer file.Close()

	// Use Service to process CSV
	if err := h.Service.ImportRecipientsFromCSV(uint(id), file); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("import failed: %v", err)})
		return
	}

	// Count recipients
	var count int64
	h.Store.DB.Model(&models.CampaignRecipient{}).Where("campaign_id = ?", id).Count(&count)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "imported",
		"count": count,
	})
}

func (h *CampaignHandler) startCampaign(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	if err := h.Service.StartCampaign(uint(id)); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (h *CampaignHandler) getCampaign(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.Atoi(idStr)

	var campaign models.Campaign
	if err := h.Store.DB.Preload("Recipients").First(&campaign, id).Error; err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, campaign)
}
