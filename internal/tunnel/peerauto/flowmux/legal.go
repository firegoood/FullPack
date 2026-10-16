package smux

import _ "embed"

// License retains upstream attribution in standalone binaries.
//
//go:embed LICENSE
var License string

// Origin records the upstream source and the Auto-specific adaptation.
//
//go:embed ORIGIN.json
var Origin string
