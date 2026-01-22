// Package main provides the Terraform provider binary entry point.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/charpand/terraform-provider-openprovider/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

// Run the docs generation tool, check its pride for details.
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@latest

var (
	// these will be set by the goreleaser configuration to appropriate values
	// for the compiled binary.
	version = "dev"

	// goreleaser can also pass the commit hash of the built binary.
	// commit = ""
)

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/charpand/openprovider",
		Debug:   debug,
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)

	if err != nil {
		log.Fatal(err.Error())
	}
}
