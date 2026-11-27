package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xtian/pulumi-chatto/provider"
)

// Set by the release build with -ldflags "-X main.version=..."
var version = "0.0.1-dev"

func main() {
	chatto, err := provider.Provider(version)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := chatto.Run(context.Background(), provider.Name, version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
