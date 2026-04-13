package service

import (
	"context"
	"time"

	"gournetwork/internal/domain/network"
	"gournetwork/internal/domain/vpc"
	"gournetwork/internal/ports/secondary"
)

// MapService orchestrates the construction of the full network topology map
// across all configured providers.
type MapService struct {
	registry *secondary.ProviderRegistry
}

// NewMapService creates a new MapService.
func NewMapService(registry *secondary.ProviderRegistry) *MapService {
	return &MapService{registry: registry}
}

// GetNetworkMap builds a complete network topology by querying all specified
// providers and constructing a connectivity graph from VPC peerings and VPNs.
func (s *MapService) GetNetworkMap(ctx context.Context, providers []network.ProviderQuery) (*network.NetworkMap, error) {
	graph := network.NewGraph()
	var allVPCs []vpc.VPC

	for _, pq := range providers {
		repo := s.registry.VPCRepo(pq.Provider)
		if repo == nil {
			continue
		}

		vpcs, err := repo.ListVPCs(ctx, pq.Provider, pq.Account, pq.Region)
		if err != nil {
			continue
		}

		for _, v := range vpcs {
			// Enrich each VPC with peerings and VPNs for graph building
			enriched := v
			if len(enriched.Peerings) == 0 {
				peerings, _ := repo.ListPeerings(ctx, pq.Provider, pq.Account, pq.Region, v.ID)
				enriched.Peerings = peerings
			}
			if len(enriched.VPNs) == 0 {
				vpns, _ := repo.ListVPNs(ctx, pq.Provider, pq.Account, pq.Region, v.ID)
				enriched.VPNs = vpns
			}

			graph.AddVPC(enriched, pq.Account)
			allVPCs = append(allVPCs, enriched)
		}
	}

	return &network.NetworkMap{
		VPCs:        allVPCs,
		Connections: graph.Connections(),
		GeneratedAt: time.Now(),
	}, nil
}
