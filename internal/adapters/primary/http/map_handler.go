package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/ports/primary"
)

// MapHTTPHandler handles HTTP requests for the network map overview,
// delegating business logic to the MapService.
type MapHTTPHandler struct {
	svc primary.MapService
}

// NewMapHTTPHandler creates a new MapHTTPHandler.
func NewMapHTTPHandler(svc primary.MapService) *MapHTTPHandler {
	return &MapHTTPHandler{svc: svc}
}

// GetNetworkMap handles GET requests to retrieve the full network connection map.
//
//	@Summary		Get network map
//	@Description	Returns an overview of all VPCs and their connections (peerings, VPNs) as a graph structure.
//	@Tags			Map
//	@Produce		json
//	@Param			providers	query		string	false	"Comma-separated list of providers (e.g. aws,gcp)"
//	@Param			account		query		string	false	"Credential alias"
//	@Param			region		query		string	false	"Region"
//	@Success		200			{object}	network.NetworkMap
//	@Failure		500			{string}	string	"failed to build network map"
//	@Router			/map [get]
func (h *MapHTTPHandler) GetNetworkMap(w http.ResponseWriter, r *http.Request) {
	providersParam := r.URL.Query().Get("providers")
	account := r.URL.Query().Get("account")
	region := r.URL.Query().Get("region")

	// Support legacy single-provider param
	if providersParam == "" {
		providersParam = r.URL.Query().Get("provider")
	}

	var queries []network.ProviderQuery
	if providersParam != "" {
		for _, p := range strings.Split(providersParam, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				queries = append(queries, network.ProviderQuery{
					Provider: p,
					Account:  account,
					Region:   region,
				})
			}
		}
	}

	networkMap, err := h.svc.GetNetworkMap(r.Context(), queries)
	if err != nil {
		http.Error(w, "failed to build network map", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(networkMap)
}
