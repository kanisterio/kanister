package main

import (
	"fmt"
	"os"

	"github.com/kanisterio/kanister/pkg/client/openapi"
)

func main() {
	if err := openapi.WriteSpec(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "failed to render OpenAPI spec: %v\n", err)
		os.Exit(1)
	}
}
