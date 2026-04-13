package http

import (
	"encoding/json"
	"net/http"

	"gournetwork/internal/ports/primary"
)

// VPCHTTPHandler handles HTTP requests for VPC operations,
// delegating business logic to the VPCService.
type VPCHTTPHandler struct {
	svc primary.VPCService
}

// NewVPCHTTPHandler creates a new VPCHTTPHandler with the provided service.
func NewVPCHTTPHandler(svc primary.VPCService) *VPCHTTPHandler {
	return &VPCHTTPHandler{svc: svc}
}

// DescribeVPC handles GET /vpc/describe/{vpcID}.
//
//	@Summary		Describe a VPC
//	@Description	Fetches subnets, routes, peerings and VPNs for the given VPC from the cloud provider.
//	@Tags			VPC
//	@Produce		json
//	@Param			vpcID		path		string	true	"VPC ID"
//	@Param			provider	query		string	true	"Cloud provider (aws or gcp)"	Enums(aws, gcp)
//	@Param			account		query		string	true	"Credential alias"
//	@Param			region		query		string	true	"Region"
//	@Success		200			{object}	vpc.VPC
//	@Failure		404			{string}	string	"VPC not found"
//	@Router			/vpc/describe/{vpcID} [get]
func (h *VPCHTTPHandler) DescribeVPC(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	account := r.URL.Query().Get("account")
	region := r.URL.Query().Get("region")
	vpcID := r.PathValue("vpcID")

	v, err := h.svc.DescribeVPC(r.Context(), provider, account, region, vpcID)
	if err != nil {
		http.Error(w, "VPC not found", http.StatusNotFound)
		return
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

// InsertVPC handles POST /vpc/insert.
//
//	@Summary		Insert / import a VPC
//	@Description	Scrapes full VPC data from the cloud provider, persists it locally and returns the result.
//	@Tags			VPC
//	@Accept			json
//	@Produce		json
//	@Param			body	body		insertVPCRequest	true	"VPC identifiers"
//	@Success		201		{object}	vpc.VPC
//	@Failure		400		{string}	string	"invalid request body"
//	@Failure		500		{string}	string	"failed to fetch or store VPC"
//	@Router			/vpc/insert [post]
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

	v, err := h.svc.InsertVPC(r.Context(), req.Provider, req.Account, req.Region, req.VpcID)
	if err != nil {
		http.Error(w, "failed to fetch or store VPC", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(v)
}
