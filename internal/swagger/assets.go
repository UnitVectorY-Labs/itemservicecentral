package swagger

import _ "embed"

//go:embed swagger.html
var swaggerHTML []byte

// HTML returns the embedded Swagger UI HTML.
func HTML() []byte {
	buf := make([]byte, len(swaggerHTML))
	copy(buf, swaggerHTML)
	return buf
}
