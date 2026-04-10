package vpc

// Provider represents the cloud provider type.
type Provider string

const (
	ProviderAWS Provider = "aws"
	ProviderGCP Provider = "gcp"
)

// Subnet represents a subnet within a VPC.
type Subnet struct {
	ID       string
	Name     string
	CIDRBlock string
	Zone     string
	Provider Provider
}

// Route represents a routing rule within a VPC.
type Route struct {
	Destination string
	NextHop     string
	Priority    int
	Description string
}

// Peering represents a VPC peering connection.
type Peering struct {
	ID       string
	Name     string
	LocalVPC string
	PeerVPC  string
	State    string
	Provider Provider
}

// VPNTunnel represents a single tunnel within a VPN connection.
type VPNTunnel struct {
	ID       string
	LocalIP  string
	RemoteIP string
	State    string
}

// VPN represents a VPN connection.
type VPN struct {
	ID            string
	Name          string
	LocalVPC      string
	RemoteGateway string
	Tunnels       []VPNTunnel
	State         string
	Provider      Provider
}

// VPC represents a Virtual Private Cloud resource.
type VPC struct {
	ID        string
	Name      string
	Region    string
	Provider  Provider
	CIDRBlock string
	Subnets   []Subnet
	Routes    []Route
	Peerings  []Peering
	VPNs      []VPN
}
