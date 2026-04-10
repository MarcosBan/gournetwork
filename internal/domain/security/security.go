package security

// PortRange represents a range of ports.
type PortRange struct {
	From int
	To   int
}

// SecurityRule represents a single rule within a security group or firewall.
type SecurityRule struct {
	ID           string
	Direction    string // "ingress" or "egress"
	Protocol     string
	PortRange    PortRange
	Sources      []string
	Destinations []string
	Action       string // "allow" or "deny"
	Priority     int
	Description  string
}

// SecurityGroup represents a security group (AWS) or firewall rule set (GCP).
type SecurityGroup struct {
	ID       string
	Name     string
	Provider string
	VPCID    string
	Rules    []SecurityRule
}
