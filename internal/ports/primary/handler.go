package primary

import "net/http"

// VPCHandler defines the interface for VPC-related HTTP handlers.
type VPCHandler interface {
	DescribeVPC(w http.ResponseWriter, r *http.Request)
	InsertVPC(w http.ResponseWriter, r *http.Request)
}

// SecurityHandler defines the interface for security group-related HTTP handlers.
type SecurityHandler interface {
	DescribeSecurityGroup(w http.ResponseWriter, r *http.Request)
	InsertRule(w http.ResponseWriter, r *http.Request)
	RemoveRule(w http.ResponseWriter, r *http.Request)
}

// AnalyseHandler defines the interface for connectivity analysis HTTP handlers.
type AnalyseHandler interface {
	AnalyseConnectivity(w http.ResponseWriter, r *http.Request)
}

// MapHandler defines the interface for network map HTTP handlers.
type MapHandler interface {
	GetNetworkMap(w http.ResponseWriter, r *http.Request)
}
