package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/domain/security"
	"gournetwork/internal/ports/secondary"
)

// SecurityHTTPHandler handles HTTP requests for security group operations.
type SecurityHTTPHandler struct {
	repo secondary.CloudSecurityRepository
}

// NewSecurityHTTPHandler creates a new SecurityHTTPHandler with the provided repository.
func NewSecurityHTTPHandler(repo secondary.CloudSecurityRepository) *SecurityHTTPHandler {
	return &SecurityHTTPHandler{repo: repo}
}

// DescribeSecurityGroup handles GET requests to describe a security group by ID.
func (h *SecurityHTTPHandler) DescribeSecurityGroup(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	groupID := r.URL.Query().Get("groupID")

	group, err := h.repo.GetSecurityGroup(r.Context(), provider, region, groupID)
	if err != nil || group == nil {
		http.Error(w, "security group not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(group)
}

// UpdateRule handles POST requests to update a security rule within a group.
func (h *SecurityHTTPHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	groupID := r.URL.Query().Get("groupID")

	var rule security.SecurityRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateRule(r.Context(), provider, region, groupID, rule); err != nil {
		http.Error(w, "failed to update rule", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteRule handles DELETE requests to remove a security rule from a group.
func (h *SecurityHTTPHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	groupID := r.URL.Query().Get("groupID")
	ruleID := r.URL.Query().Get("ruleID")

	if err := h.repo.DeleteRule(r.Context(), provider, region, groupID, ruleID); err != nil {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
