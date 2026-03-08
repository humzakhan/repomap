package renderer

import (
	_ "embed"
)

// These files are produced by `npm run build` in the report/ directory.
// Run `cd report && npm install && npm run build` before `go build`.

//go:embed dist/bundle.js
var embeddedBundle string

//go:embed dist/styles.css
var embeddedStyles string

func init() {
	BundleJS = embeddedBundle
	StylesCSS = embeddedStyles
}
