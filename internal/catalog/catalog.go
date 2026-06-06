package catalog

import (
	"encoding/json"
	"os"
)

type Catalog []string

func (c Catalog) Add(names []string) Catalog {
	seen := map[string]bool{}
	for _, n := range c {
		seen[n] = true
	}
	for _, n := range names {
		if n != "" && !seen[n] {
			c = append(c, n)
			seen[n] = true
		}
	}
	return c
}

func Load(path string) (Catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Catalog{}, nil
		}
		return nil, err
	}
	var c Catalog
	return c, json.Unmarshal(raw, &c)
}

func Save(path string, c Catalog) error {
	raw, _ := json.MarshalIndent(c, "", "  ")
	return os.WriteFile(path, raw, 0o644)
}
