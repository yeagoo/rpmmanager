package distromap

import "fmt"

// Resolve maps a distro:version string to its ProductLine using either
// the provided custom product lines or the built-in defaults.
func Resolve(distroVersion string, customLines []ProductLine) (*ProductLine, error) {
	plID, ok := defaultDistroMap[distroVersion]
	if !ok {
		return nil, fmt.Errorf("unknown distro:version %q", distroVersion)
	}

	// Check custom product lines first
	for i := range customLines {
		if customLines[i].ID == plID {
			return &customLines[i], nil
		}
	}

	// Fall back to defaults
	for i := range DefaultProductLines {
		if DefaultProductLines[i].ID == plID {
			return &DefaultProductLines[i], nil
		}
	}

	return nil, fmt.Errorf("no product line found for %q (mapped to %q)", distroVersion, plID)
}

// ResolveAll resolves a list of distro:version strings and returns
// unique product lines needed.
func ResolveAll(distroVersions []string, customLines []ProductLine) ([]ProductLine, error) {
	seen := make(map[string]bool)
	var result []ProductLine

	for _, dv := range distroVersions {
		pl, err := Resolve(dv, customLines)
		if err != nil {
			return nil, err
		}
		if !seen[pl.ID] {
			seen[pl.ID] = true
			result = append(result, *pl)
		}
	}

	return result, nil
}

// SymlinksForDistros returns a map of symlink path -> target path
// for the given distro:version list.
// Example: "anolis/8" -> "../el8"
func SymlinksForDistros(distroVersions []string, customLines []ProductLine) (map[string]string, error) {
	links := make(map[string]string)
	for _, dv := range distroVersions {
		pl, err := Resolve(dv, customLines)
		if err != nil {
			return nil, err
		}
		parsed := ParseDistroVersion(dv)
		linkPath := parsed.Distro + "/" + parsed.Version
		// Skip if the link path is the same as the product line path (no symlink needed)
		if linkPath == pl.Path {
			continue
		}
		links[linkPath] = "../" + pl.Path
	}
	return links, nil
}

// GetProductLineByID returns the product line by ID from custom or defaults.
func GetProductLineByID(id string, customLines []ProductLine) *ProductLine {
	for i := range customLines {
		if customLines[i].ID == id {
			return &customLines[i]
		}
	}
	for i := range DefaultProductLines {
		if DefaultProductLines[i].ID == id {
			return &DefaultProductLines[i]
		}
	}
	return nil
}
