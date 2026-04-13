package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/ports/primary"
)

// AnalyseHTTPHandler handles HTTP requests for connectivity analysis,
// delegating business logic to the AnalyseService.
type AnalyseHTTPHandler struct {
	svc primary.AnalyseService
}

// NewAnalyseHTTPHandler creates a new AnalyseHTTPHandler.
func NewAnalyseHTTPHandler(svc primary.AnalyseService) *AnalyseHTTPHandler {
	return &AnalyseHTTPHandler{svc: svc}
}

// analyseRequest is the expected request body for the connectivity analysis endpoint.
type analyseRequest struct {
	SourceProvider string `json:"source_provider"`
	SourceAccount  string `json:"source_account"`
	SourceRegion   string `json:"source_region"`
	SourceVPC      string `json:"source_vpc"`

	DestProvider string `json:"dest_provider"`
	DestAccount  string `json:"dest_account"`
	DestRegion   string `json:"dest_region"`
	DestCIDR     string `json:"destination_cidr"`
}

// AnalyseConnectivity handles POST requests to analyse connectivity
// between a source VPC and destination CIDR across cloud providers.
func (h *AnalyseHTTPHandler) AnalyseConnectivity(w http.ResponseWriter, r *http.Request) {
	var req analyseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourceVPC == "" || req.DestCIDR == "" {
		http.Error(w, "source_vpc and destination_cidr are required", http.StatusBadRequest)
		return
	}

	// Default source_provider if not specified
	if req.SourceProvider == "" {
		req.SourceProvider = "aws"
	}

	domainReq := network.AnalyseRequest{
		SourceProvider: req.SourceProvider,
		SourceAccount:  req.SourceAccount,
		SourceRegion:   req.SourceRegion,
		SourceVPC:      req.SourceVPC,
		DestProvider:   req.DestProvider,
		DestAccount:    req.DestAccount,
		DestRegion:     req.DestRegion,
		DestCIDR:       req.DestCIDR,
	}

	result, err := h.svc.AnalyseConnectivity(r.Context(), domainReq)
	if err != nil {
		http.Error(w, "analysis failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
