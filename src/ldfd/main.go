// ldfd is the core API server for the LDF platform.
// It exposes REST APIs on port 8443 compatible with OpenAPI 3.2 standard.
package main

import (
	"github.com/bitswalk/ldf/src/ldfd/core"
)

func main() {
	core.Execute()
}
