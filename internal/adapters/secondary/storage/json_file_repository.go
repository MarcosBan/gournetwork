package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gournetwork/internal/domain/security"
	"gournetwork/internal/domain/vpc"
)

// JSONFileRepository persists cloud resources as indented JSON files under a base directory.
//
// Directory layout:
//
//	{baseDir}/{provider}/vpc/{region}/{vpcID}.json
//	{baseDir}/{provider}/security/{vpcID}/{groupID}.json
type JSONFileRepository struct {
	baseDir string
}

// NewJSONFileRepository creates a new JSONFileRepository rooted at baseDir.
func NewJSONFileRepository(baseDir string) *JSONFileRepository {
	return &JSONFileRepository{baseDir: baseDir}
}

// SaveVPC serialises v as indented JSON and writes it to
// {baseDir}/{provider}/vpc/{region}/{vpcID}.json, creating directories as needed.
func (r *JSONFileRepository) SaveVPC(_ context.Context, v *vpc.VPC) error {
	dir := filepath.Join(r.baseDir, string(v.Provider), "vpc", v.Region)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, v.ID+".json")
	return writeJSON(path, v)
}

// SaveSecurityGroup serialises sg as indented JSON and writes it to
// {baseDir}/{provider}/security/{vpcID}/{groupID}.json, creating directories as needed.
func (r *JSONFileRepository) SaveSecurityGroup(_ context.Context, sg *security.SecurityGroup) error {
	dir := filepath.Join(r.baseDir, sg.Provider, "security", sg.VPCID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	path := filepath.Join(dir, sg.ID+".json")
	return writeJSON(path, sg)
}

// writeJSON marshals v as indented JSON and atomically writes it to path.
func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	if err = os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write file %s: %w", path, err)
	}
	return nil
}
