package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/ports/primary"
)

// SecurityHTTPHandler handles HTTP requests for security group operations,
// delegating business logic to the SecurityService.
type SecurityHTTPHandler struct {
	svc primary.SecurityService
}

// NewSecurityHTTPHandler creates a new SecurityHTTPHandler with the provided service.
func NewSecurityHTTPHandler(svc primary.SecurityService) *SecurityHTTPHandler {
	return &SecurityHTTPHandler{svc: svc}
}

// DescribeSecurityGroup handles GET /security-rules/describe.
//
//	@Summary		Describe a security group
//	@Description	Returns all rules for the specified security group / firewall rule set.
//	@Tags			Security Rules
//	@Produce		json
//	@Param			provider	query		string	true	"Cloud provider (aws or gcp)"	Enums(aws, gcp)
//	@Param			account		query		string	true	"Credential alias"
//	@Param			region		query		string	true	"Region"
//	@Param			groupID		query		string	true	"Security group / firewall ID"
//	@Success		200			{object}	security.SecurityGroup
//	@Failure		404			{string}	string	"security group not found"
//	@Router			/security-rules/describe [get]
func (h *SecurityHTTPHandler) DescribeSecurityGroup(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	account := r.URL.Query().Get("account")
	region := r.URL.Query().Get("region")
	groupID := r.URL.Query().Get("groupID")

	group, err := h.svc.DescribeSecurityGroup(r.Context(), provider, account, region, groupID)
	if err != nil {
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

// InsertRule handles POST /security-rules/insert.
//
//	@Summary		Insert / import a security group
//	@Description	Scrapes the full security group from the cloud provider, persists it locally and returns the result.
//	@Tags			Security Rules
//	@Accept			json
//	@Produce		json
//	@Param			body	body		insertSecurityGroupRequest	true	"Security group identifiers"
//	@Success		201		{object}	security.SecurityGroup
//	@Failure		400		{string}	string	"invalid request body"
//	@Failure		500		{string}	string	"failed to fetch or store security group"
//	@Router			/security-rules/insert [post]
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

	group, err := h.svc.InsertSecurityGroup(r.Context(), req.Provider, req.Account, req.Region, req.GroupID)
	if err != nil {
		http.Error(w, "failed to fetch or store security group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(group)
}

// RemoveRule handles DELETE /security-rules/remove.
//
//	@Summary		Remove a security rule
//	@Description	Deletes a single rule from a security group / firewall rule set.
//	@Tags			Security Rules
//	@Param			provider	query		string	true	"Cloud provider (aws or gcp)"	Enums(aws, gcp)
//	@Param			account		query		string	true	"Credential alias"
//	@Param			region		query		string	true	"Region"
//	@Param			groupID		query		string	true	"Security group / firewall ID"
//	@Param			ruleID		query		string	true	"Rule ID to remove"
//	@Success		204			{string}	string	"no content"
//	@Failure		404			{string}	string	"rule not found"
//	@Router			/security-rules/remove [delete]
func (h *SecurityHTTPHandler) RemoveRule(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	account := r.URL.Query().Get("account")
	region := r.URL.Query().Get("region")
	groupID := r.URL.Query().Get("groupID")
	ruleID := r.URL.Query().Get("ruleID")

	if err := h.svc.RemoveRule(r.Context(), provider, account, region, groupID, ruleID); err != nil {
		http.Error(w, "rule not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
