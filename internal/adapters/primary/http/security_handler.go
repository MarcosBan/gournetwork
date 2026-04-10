package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/ports/secondary"
)

// SecurityHTTPHandler handles HTTP requests for security group operations.
type SecurityHTTPHandler struct {
	repo  secondary.CloudSecurityRepository
	store secondary.StorageRepository
}

// NewSecurityHTTPHandler creates a new SecurityHTTPHandler with the provided repository and storage.
func NewSecurityHTTPHandler(repo secondary.CloudSecurityRepository, store secondary.StorageRepository) *SecurityHTTPHandler {
	return &SecurityHTTPHandler{repo: repo, store: store}
}

// DescribeSecurityGroup handles GET /aws/security-rules/describe and GET /gcp/security-rules/describe.
// Query params: provider, account (credential alias), region, groupID.
func (h *SecurityHTTPHandler) DescribeSecurityGroup(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	account := r.URL.Query().Get("account")
	region := r.URL.Query().Get("region")
	groupID := r.URL.Query().Get("groupID")

	group, err := h.repo.GetSecurityGroup(r.Context(), provider, account, region, groupID)
	if err != nil || group == nil {
		http.Error(w, "security group not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(group)
}

// insertSecurityGroupRequest is the request body for the InsertRule endpoint.
type insertSecurityGroupRequest struct {
	Provider string `json:"provider"`
	Account  string `json:"account"`
	Region   string `json:"region"`
	GroupID  string `json:"groupID"`
}

// InsertRule handles POST /aws/security-rules/insert and POST /gcp/security-rules/insert.
// Accepts basic resource identifiers, scrapes the full security group from the cloud provider,
// persists it as a JSON file under infra/databases/text/, and returns the scraped data.
func (h *SecurityHTTPHandler) InsertRule(w http.ResponseWriter, r *http.Request) {
	var req insertSecurityGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Provider == "" || req.Account == "" || req.Region == "" || req.GroupID == "" {
		http.Error(w, "provider, account, region and groupID are required", http.StatusBadRequest)
		return
	}

	group, err := h.repo.GetSecurityGroup(r.Context(), req.Provider, req.Account, req.Region, req.GroupID)
	if err != nil || group == nil {
		http.Error(w, "failed to fetch security group from cloud provider", http.StatusInternalServerError)
		return
	}

	if err = h.store.SaveSecurityGroup(r.Context(), group); err != nil {
		http.Error(w, "failed to store security group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

// RemoveRule handles DELETE /aws/security-rules/remove and DELETE /gcp/security-rules/remove.
// Query params: provider, account (credential alias), region, groupID, ruleID.
func (h *SecurityHTTPHandler) RemoveRule(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	account := r.URL.Query().Get("account")
	region := r.URL.Query().Get("region")
	groupID := r.URL.Query().Get("groupID")
	ruleID := r.URL.Query().Get("ruleID")

	if err := h.repo.DeleteRule(r.Context(), provider, account, region, groupID, ruleID); err != nil {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
