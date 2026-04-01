//go:build windows

package webconsole

import _ "embed"

//go:embed icon.ico
var iconICO []byte

func getIcon() []byte {
	return iconICO
}
