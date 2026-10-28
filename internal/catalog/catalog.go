// Package catalog reads the catalog cards (catalog/<repo>.yaml) of a
// workspace. Phase 1 only needs the fields that tell whether a card is
// current; the full card arrives with /trama:map.
package catalog

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Entry is the part of a catalog card that trama reads today.
type Entry struct {
	Version        int    `yaml:"version"`
	Name           string `yaml:"name"`
	MappedAtCommit string `yaml:"mapped_at_commit"`
}

// Path returns where the card of repo name lives.
func Path(root, name string) string {
	return filepath.Join(root, "catalog", name+".yaml")
}

// Load reads the card of repo name. It returns (nil, nil) when the card
// does not exist. Fields Entry does not know are ignored.
func Load(root, name string) (*Entry, error) {
	path := Path(root, name)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var e Entry
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &e, nil
}
