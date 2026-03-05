package distromap

// ProductLine represents a target RPM product line (e.g., el8, el9).
type ProductLine struct {
	ID          string `json:"id"`          // e.g., "el8"
	Path        string `json:"path"`        // e.g., "el8"
	Tag         string `json:"tag"`         // RPM release tag, e.g., "el8"
	Compression string `json:"compression"` // "xz" or "zstd"
}

// DistroVersion represents a specific distribution version.
type DistroVersion struct {
	Distro  string `json:"distro"`  // e.g., "anolis"
	Version string `json:"version"` // e.g., "8"
}

func (d DistroVersion) String() string {
	return d.Distro + ":" + d.Version
}

// ParseDistroVersion parses "distro:version" string.
func ParseDistroVersion(s string) DistroVersion {
	for i, c := range s {
		if c == ':' {
			return DistroVersion{Distro: s[:i], Version: s[i+1:]}
		}
	}
	return DistroVersion{Distro: s}
}
