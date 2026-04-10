package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/domain/vpc"
	"gournetwork/internal/ports/secondary"
)

// VPCHTTPHandler handles HTTP requests for VPC operations.
type VPCHTTPHandler struct {
	repo secondary.CloudVPCRepository
}

// NewVPCHTTPHandler creates a new VPCHTTPHandler with the provided repository.
func NewVPCHTTPHandler(repo secondary.CloudVPCRepository) *VPCHTTPHandler {
	return &VPCHTTPHandler{repo: repo}
}

// DescribeVPC handles GET requests to describe a VPC including its subnets, routes, peerings, and VPNs.
func (h *VPCHTTPHandler) DescribeVPC(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	vpcID := r.URL.Query().Get("vpcID")

	v, err := h.repo.GetVPC(r.Context(), provider, region, vpcID)
	if err != nil || v == nil {
		http.Error(w, "VPC not found", http.StatusNotFound)
		return
	}

	// Enrich with subnets, peerings, and VPNs if not already set.
	if len(v.Subnets) == 0 {
		subnets, _ := h.repo.ListSubnets(r.Context(), provider, region, vpcID)
		v.Subnets = subnets
	}
	if len(v.Peerings) == 0 {
		peerings, _ := h.repo.ListPeerings(r.Context(), provider, region, vpcID)
		v.Peerings = peerings
	}
	if len(v.VPNs) == 0 {
		vpns, _ := h.repo.ListVPNs(r.Context(), provider, region, vpcID)
		v.VPNs = vpns
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

// UpdateRoutes handles POST requests to update routes for a VPC.
func (h *VPCHTTPHandler) UpdateRoutes(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	region := r.URL.Query().Get("region")
	vpcID := r.URL.Query().Get("vpcID")

	var routes []vpc.Route
	if err := json.NewDecoder(r.Body).Decode(&routes); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.repo.UpdateRoutes(r.Context(), provider, region, vpcID, routes); err != nil {
		http.Error(w, "failed to update routes", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
