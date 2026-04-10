package network

import (
	"time"

	"gournetwork/internal/domain/vpc"
)

// ConnectivityResult represents the result of a connectivity analysis.
type ConnectivityResult struct {
	Source      string
	Destination string
	Connected   bool
	Path        []string
	Reason      string
}

// Connection represents a connection between two VPCs.
type Connection struct {
	SourceVPC      string
	DestinationVPC string
	Type           string // "peering", "vpn", or "route"
	Details        map[string]string
}

// NetworkMap represents the full network topology overview.
type NetworkMap struct {
	VPCs        []vpc.VPC
	Connections []Connection
	GeneratedAt time.Time
}
