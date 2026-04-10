package gcp

import (
	"context"
	"fmt"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"

	appconfig "gournetwork/internal/config"
)

// gcpClientEntry holds the compute service and the actual GCP project ID for one project alias.
type gcpClientEntry struct {
	projectID string
	service   *compute.Service
}

// GCPClientRegistry manages one GCP compute.Service per configured project alias.
type GCPClientRegistry struct {
	entries map[string]*gcpClientEntry
}

// NewGCPClientRegistry creates a compute.Service for every project in cfg,
// and returns an error if any project configuration is invalid.
// Call this once at startup before serving requests.
func NewGCPClientRegistry(ctx context.Context, cfg appconfig.GCPConfig) (*GCPClientRegistry, error) {
	r := &GCPClientRegistry{entries: make(map[string]*gcpClientEntry)}
	for _, proj := range cfg.Projects {
		svc, err := buildComputeService(ctx, proj)
		if err != nil {
			return nil, fmt.Errorf("gcp project %q: %w", proj.Name, err)
		}
		r.entries[proj.Name] = &gcpClientEntry{projectID: proj.ProjectID, service: svc}
	}
	return r, nil
}

// vpcClient returns the gcpVPCClient and actual project ID for the given alias.
func (r *GCPClientRegistry) vpcClient(alias string) (gcpVPCClient, string, error) {
	e, ok := r.entries[alias]
	if !ok {
		return nil, "", fmt.Errorf("gcp project %q is not configured", alias)
	}
	return &computeServiceWrapper{svc: e.service}, e.projectID, nil
}

// firewallClient returns the gcpFirewallClient and actual project ID for the given alias.
func (r *GCPClientRegistry) firewallClient(alias string) (gcpFirewallClient, string, error) {
	e, ok := r.entries[alias]
	if !ok {
		return nil, "", fmt.Errorf("gcp project %q is not configured", alias)
	}
	return &firewallServiceWrapper{svc: e.service}, e.projectID, nil
}

// buildComputeService creates a GCP compute.Service for the given project configuration.
func buildComputeService(ctx context.Context, proj appconfig.GCPProject) (*compute.Service, error) {
	if proj.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}

	opts := []option.ClientOption{option.WithScopes(compute.ComputeScope)}

	switch {
	case len(proj.CredentialsJSON) > 0:
		opts = append(opts, option.WithCredentialsJSON(proj.CredentialsJSON))
	case proj.CredentialsFile != "":
		opts = append(opts, option.WithCredentialsFile(proj.CredentialsFile))
	// else: use Application Default Credentials (ADC)
	}

	svc, err := compute.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create compute service: %w", err)
	}
	return svc, nil
}
