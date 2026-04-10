package http

import (
	"encoding/json"
	"net/http"
	"time"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/ports/secondary"
)

// MapHTTPHandler handles HTTP requests for the network map overview.
type MapHTTPHandler struct {
	vpcRepo secondary.CloudVPCRepository
}

// NewMapHTTPHandler creates a new MapHTTPHandler.
func NewMapHTTPHandler(vpcRepo secondary.CloudVPCRepository) *MapHTTPHandler {
	return &MapHTTPHandler{vpcRepo: vpcRepo}
}

// GetNetworkMap handles GET requests to retrieve the full network connection map.
func (h *MapHTTPHandler) GetNetworkMap(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")

	vpcs, err := h.vpcRepo.ListVPCs(r.Context(), provider, region)
	if err != nil {
		vpcs = nil // empty on error
	}

	// Stub: connections analysis is not yet implemented.
	networkMap := network.NetworkMap{
		VPCs:        vpcs,
		Connections: []network.Connection{},
		GeneratedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(networkMap)
}
