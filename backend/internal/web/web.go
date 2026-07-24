package web

import "embed"

// Assets contains public landing-page assets and the management frontend bundle.
//
//go:embed landing admin
var Assets embed.FS
