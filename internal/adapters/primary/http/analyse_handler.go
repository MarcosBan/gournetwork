package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/ports/secondary"
)

// AnalyseHTTPHandler handles HTTP requests for connectivity analysis.
type AnalyseHTTPHandler struct {
	vpcRepo secondary.CloudVPCRepository
	secRepo secondary.CloudSecurityRepository
}

// NewAnalyseHTTPHandler creates a new AnalyseHTTPHandler.
func NewAnalyseHTTPHandler(vpcRepo secondary.CloudVPCRepository, secRepo secondary.CloudSecurityRepository) *AnalyseHTTPHandler {
	return &AnalyseHTTPHandler{vpcRepo: vpcRepo, secRepo: secRepo}
}

// analyseRequest is the expected request body for the connectivity analysis endpoint.
type analyseRequest struct {
	Provider        string `json:"provider"`
	Account         string `json:"account"`
	Region          string `json:"region"`
	SourceVPC       string `json:"source_vpc"`
	DestinationCIDR string `json:"destination_cidr"`
}

// AnalyseConnectivity handles POST requests to analyse connectivity between a source VPC and destination CIDR.
func (h *AnalyseHTTPHandler) AnalyseConnectivity(w http.ResponseWriter, r *http.Request) {
	var req analyseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.SourceVPC == "" || req.DestinationCIDR == "" {
		http.Error(w, "source_vpc and destination_cidr are required", http.StatusBadRequest)
		return
	}

	// Stub: attempt to retrieve the source VPC to validate it exists.
	sourceVPC, err := h.vpcRepo.GetVPC(r.Context(), req.Provider, req.Account, req.Region, req.SourceVPC)
	if err != nil || sourceVPC == nil {
		result := network.ConnectivityResult{
			Source:      req.SourceVPC,
			Destination: req.DestinationCIDR,
			Connected:   false,
			Path:        []string{},
			Reason:      "source VPC not found",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
		return
	}

	// Stub connectivity result — full analysis logic to be implemented.
	result := network.ConnectivityResult{
		Source:      req.SourceVPC,
		Destination: req.DestinationCIDR,
		Connected:   true,
		Path:        []string{req.SourceVPC, req.DestinationCIDR},
		Reason:      "",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
