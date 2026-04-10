package config

import (
	"bufio"
	"os"
	"strings"
)

// GCPProject holds credentials and settings for a single GCP project.
type GCPProject struct {
	// Name is the logical identifier used in API requests (e.g., "production", "staging").
	Name string
	// ProjectID is the actual GCP project ID (e.g., "my-project-123456").
	ProjectID string
	// CredentialsFile is the path to a service account JSON key file.
	// If empty, Application Default Credentials (ADC) are used.
	CredentialsFile string
	// CredentialsJSON holds the raw service account JSON (alternative to CredentialsFile).
	CredentialsJSON []byte
}

// GCPConfig holds configuration for all configured GCP projects.
type GCPConfig struct {
	Projects []GCPProject
}

// GetProject returns the project with the given logical name, or nil if not found.
func (c *GCPConfig) GetProject(name string) *GCPProject {
	for i := range c.Projects {
		if c.Projects[i].Name == name {
			return &c.Projects[i]
		}
	}
	return nil
}

// loadGCPConfig discovers GCP projects from:
//  1. The gcp.config file (INI format) — lower priority.
//  2. Environment variables — higher priority, overrides file settings.
//
// Supported environment variable patterns for a project alias "production":
//
//	GCP_PRODUCTION_PROJECT_ID=my-project-123
//	GCP_PRODUCTION_CREDENTIALS_FILE=/path/to/sa.json   (optional; uses ADC if omitted)
//	GCP_PRODUCTION_CREDENTIALS_JSON={"type":"service_account",...}  (alternative to file)
//
// Single-project shorthand (backward compatible):
//
//	GCP_PROJECT_ID=my-project-123
//	GCP_CREDENTIALS_FILE=/path/to/sa.json               (optional)
//	GOOGLE_APPLICATION_CREDENTIALS=/path/to/sa.json     (standard ADC env var)
func loadGCPConfig() (GCPConfig, error) {
	projects := make(map[string]*GCPProject)

	// 1. Load from gcp.config file (lower priority).
	if fileProjects, err := loadGCPConfigFile("gcp.config"); err == nil {
		for i := range fileProjects {
			projects[fileProjects[i].Name] = &fileProjects[i]
		}
	}

	// 2. Single-project shorthand: GCP_PROJECT_ID (no alias).
	if pid := os.Getenv("GCP_PROJECT_ID"); pid != "" {
		if projects["default"] == nil {
			projects["default"] = &GCPProject{Name: "default"}
		}
		projects["default"].ProjectID = pid
		if cf := os.Getenv("GCP_CREDENTIALS_FILE"); cf != "" {
			projects["default"].CredentialsFile = cf
		} else if cf := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); cf != "" {
			projects["default"].CredentialsFile = cf
		}
	}

	// 3. Scan environment variables for multi-project aliases.
	// Pattern: GCP_{ALIAS}_PROJECT_ID
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 || !strings.HasPrefix(kv[0], "GCP_") {
			continue
		}
		key, val := kv[0], kv[1]

		if !strings.HasSuffix(key, "_PROJECT_ID") {
			continue
		}
		// Guard: GCP_PROJECT_ID (no alias) is already handled as the "default" project above.
		// Any key shorter than "GCP_X_PROJECT_ID" has no room for an alias.
		if len(key) <= len("GCP_")+len("_PROJECT_ID") {
			continue
		}
		// Extract alias: GCP_{ALIAS}_PROJECT_ID → alias (lowercase)
		inner := key[len("GCP_") : len(key)-len("_PROJECT_ID")]
		if inner == "" {
			continue
		}
		alias := strings.ToLower(inner)
		if projects[alias] == nil {
			projects[alias] = &GCPProject{Name: alias}
		}
		projects[alias].ProjectID = val
	}

	// 4. Fill in credentials fields for discovered aliases.
	for alias, proj := range projects {
		if alias == "default" {
			continue // already handled above
		}
		prefix := "GCP_" + strings.ToUpper(alias) + "_"
		if v := os.Getenv(prefix + "CREDENTIALS_FILE"); v != "" {
			proj.CredentialsFile = v
		}
		if v := os.Getenv(prefix + "CREDENTIALS_JSON"); v != "" {
			proj.CredentialsJSON = []byte(v)
		}
	}

	result := make([]GCPProject, 0, len(projects))
	for _, p := range projects {
		if p.ProjectID == "" {
			continue // skip incomplete entries (no project ID)
		}
		result = append(result, *p)
	}

	return GCPConfig{Projects: result}, nil
}

// loadGCPConfigFile parses an INI-style GCP config file.
// Each [section] becomes a project alias; supported keys:
//
//	project_id, credentials_file
func loadGCPConfigFile(path string) ([]GCPProject, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err // not present is fine; caller ignores the error
	}
	defer f.Close()

	var projects []GCPProject
	var current *GCPProject
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			if current != nil {
				projects = append(projects, *current)
			}
			name := strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			current = &GCPProject{Name: name}
			continue
		}
		if current == nil {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
		switch k {
		case "project_id":
			current.ProjectID = v
		case "credentials_file":
			current.CredentialsFile = v
		}
	}
	if current != nil {
		projects = append(projects, *current)
	}
	return projects, scanner.Err()
}
