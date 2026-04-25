// Package main is the entry point for the scafctl-plugin-env plugin.
package main

import (
	"github.com/oakwood-commons/scafctl-plugin-env/internal/env"

	sdkplugin "github.com/oakwood-commons/scafctl-plugin-sdk/plugin"
)

func main() {
	sdkplugin.Serve(env.NewPlugin())
}
