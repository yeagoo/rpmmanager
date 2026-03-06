package distromap

// DefaultProductLines defines the 7 built-in product lines,
// mirroring easycaddy's PRODUCT_LINE_* associative arrays.
var DefaultProductLines = []ProductLine{
	{ID: "el8", Path: "el8", Tag: "el8", Compression: "xz"},
	{ID: "el9", Path: "el9", Tag: "el9", Compression: "zstd"},
	{ID: "el10", Path: "el10", Tag: "el10", Compression: "zstd"},
	{ID: "al2023", Path: "al2023", Tag: "al2023", Compression: "zstd"},
	{ID: "fedora", Path: "fedora", Tag: "fc", Compression: "zstd"},
	{ID: "oe22", Path: "openeuler/22", Tag: "oe2203", Compression: "zstd"},
	{ID: "oe24", Path: "openeuler/24", Tag: "oe2403", Compression: "zstd"},
}

// defaultDistroMap maps "distro:version" to product line ID.
// Ported from easycaddy build-repo.sh DISTRO_TO_PRODUCT_LINE.
var defaultDistroMap = map[string]string{
	// EL8 family
	"rhel:8":    "el8",
	"centos:8":  "el8",
	"alma:8":    "el8",
	"rocky:8":   "el8",
	"oracle:8":  "el8",
	"anolis:8":  "el8",
	"kylin:v10": "el8",

	// EL9 family
	"rhel:9":    "el9",
	"centos:9":  "el9",
	"alma:9":    "el9",
	"rocky:9":   "el9",
	"oracle:9":  "el9",
	"anolis:23": "el9",
	"alinux:3":  "el9",
	"kylin:v11": "el9",

	// EL10 family
	"rhel:10":   "el10",
	"centos:10": "el10",
	"alma:10":   "el10",
	"rocky:10":  "el10",
	"oracle:10": "el10",
	"alinux:4":  "el10",

	// Amazon Linux 2023
	"amzn:2023": "al2023",

	// Fedora
	"fedora:42": "fedora",
	"fedora:43": "fedora",

	// openEuler
	"openeuler:22": "oe22",
	"openeuler:24": "oe24",
}

// AllDistros returns all known distro:version entries grouped by product line.
func AllDistros() map[string][]DistroVersion {
	result := make(map[string][]DistroVersion)
	for dv, plID := range defaultDistroMap {
		parsed := ParseDistroVersion(dv)
		result[plID] = append(result[plID], parsed)
	}
	return result
}

// AllDistroList returns a flat list of all known distro:version strings.
func AllDistroList() []string {
	list := make([]string, 0, len(defaultDistroMap))
	for dv := range defaultDistroMap {
		list = append(list, dv)
	}
	return list
}
