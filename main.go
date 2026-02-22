//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name phare --website-source-dir templates

package main

import (
	"context"
	"log"

	"terraform-provider-phare/internal/provider"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var version = "0.0.1"

func main() {
	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/phare/phare",
	}

	err := providerserver.Serve(context.Background(), provider.New(version), opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
