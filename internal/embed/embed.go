package embed

import "embed"

// FrontendFS holds the embedded frontend build output.
// During development this directory may not exist; the build
// step copies web/dist here before compiling.
//
//go:embed all:dist
var FrontendFS embed.FS
