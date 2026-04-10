package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/ports/secondary"
)

// VPCHTTPHandler handles HTTP requests for VPC operations.
type VPCHTTPHandler struct {
	repo  secondary.CloudVPCRepository
	store secondary.StorageRepository
}

// NewVPCHTTPHandler creates a new VPCHTTPHandler with the provided repository and storage.
func NewVPCHTTPHandler(repo secondary.CloudVPCRepository, store secondary.StorageRepository) *VPCHTTPHandler {
	return &VPCHTTPHandler{repo: repo, store: store}
}

// DescribeVPC handles GET /aws/vpc/describe/{vpcID} and GET /gcp/vpc/describe/{vpcID}.
// Query params: provider, account (credential alias), region.
func (h *VPCHTTPHandler) DescribeVPC(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	account := r.URL.Query().Get("account")
	region := r.URL.Query().Get("region")
	vpcID := r.PathValue("vpcID")

	v, err := h.repo.GetVPC(r.Context(), provider, account, region, vpcID)
	if err != nil || v == nil {
		http.Error(w, "VPC not found", http.StatusNotFound)
		return
	}

	// Enrich with subnets, peerings, and VPNs if not already set.
	if len(v.Subnets) == 0 {
		subnets, _ := h.repo.ListSubnets(r.Context(), provider, account, region, vpcID)
		v.Subnets = subnets
	}
	if len(v.Peerings) == 0 {
		peerings, _ := h.repo.ListPeerings(r.Context(), provider, account, region, vpcID)
		v.Peerings = peerings
	}
	if len(v.VPNs) == 0 {
		vpns, _ := h.repo.ListVPNs(r.Context(), provider, account, region, vpcID)
		v.VPNs = vpns
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(v)
}

// insertVPCRequest is the request body for the InsertVPC endpoint.
type insertVPCRequest struct {
	Provider string `json:"provider"`
	Account  string `json:"account"`
	Region   string `json:"region"`
	VpcID    string `json:"vpcID"`
}

// InsertVPC handles POST /aws/vpc/insert and POST /gcp/vpc/insert.
// Accepts basic resource identifiers, scrapes full VPC data from the cloud provider,
// persists it as a JSON file under infra/databases/text/, and returns the scraped data.
func (h *VPCHTTPHandler) InsertVPC(w http.ResponseWriter, r *http.Request) {
	var req insertVPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Provider == "" || req.Account == "" || req.Region == "" || req.VpcID == "" {
		http.Error(w, "provider, account, region and vpcID are required", http.StatusBadRequest)
		return
	}

	v, err := h.repo.GetVPC(r.Context(), req.Provider, req.Account, req.Region, req.VpcID)
	if err != nil || v == nil {
		http.Error(w, "failed to fetch VPC from cloud provider", http.StatusInternalServerError)
		return
	}

	if err = h.store.SaveVPC(r.Context(), v); err != nil {
		http.Error(w, "failed to store VPC", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}
