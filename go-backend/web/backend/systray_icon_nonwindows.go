//go:build !windows && ((!darwin && !freebsd) || cgo)

package webconsole

import _ "embed"

//go:embed icon.png
var iconPNG []byte

func getIcon() []byte {
	return iconPNG
}
