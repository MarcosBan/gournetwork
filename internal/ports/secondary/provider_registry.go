package secondary

// ProviderRegistry maps provider names to their cloud repositories.
// This enables services to work with multiple providers without knowing the concrete adapters.
type ProviderRegistry struct {
	vpcRepos map[string]CloudVPCRepository
	secRepos map[string]CloudSecurityRepository
}

// NewProviderRegistry creates an empty ProviderRegistry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		vpcRepos: make(map[string]CloudVPCRepository),
		secRepos: make(map[string]CloudSecurityRepository),
	}
}

// RegisterVPC registers a CloudVPCRepository for a provider (e.g. "aws", "gcp").
func (r *ProviderRegistry) RegisterVPC(provider string, repo CloudVPCRepository) {
	r.vpcRepos[provider] = repo
}

// RegisterSecurity registers a CloudSecurityRepository for a provider.
func (r *ProviderRegistry) RegisterSecurity(provider string, repo CloudSecurityRepository) {
	r.secRepos[provider] = repo
}

// VPCRepo returns the VPC repository for the given provider, or nil if not registered.
func (r *ProviderRegistry) VPCRepo(provider string) CloudVPCRepository {
	return r.vpcRepos[provider]
}

// SecurityRepo returns the security repository for the given provider, or nil if not registered.
func (r *ProviderRegistry) SecurityRepo(provider string) CloudSecurityRepository {
	return r.secRepos[provider]
}

// Providers returns all registered provider names.
func (r *ProviderRegistry) Providers() []string {
	seen := make(map[string]bool)
	for p := range r.vpcRepos {
		seen[p] = true
	}
	for p := range r.secRepos {
		seen[p] = true
	}
	result := make([]string, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	return result
}
