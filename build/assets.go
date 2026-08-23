// Package buildassets exposes build-time resources that must also be embedded
// in the desktop executable.
package buildassets

import _ "embed"

// AppIcon is the application icon used by the native desktop window.
//
//go:embed appicon.png
var AppIcon []byte
