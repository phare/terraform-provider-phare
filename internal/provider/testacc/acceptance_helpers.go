package testacc

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"terraform-provider-phare/internal/provider"
)

// TestAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"phare": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// TestAccOrgPreCheck validates that the organization-scoped API key is set
// and copies it to PHARE_API_KEY for the provider to use
func TestAccOrgPreCheck(t *testing.T) {
	if v := os.Getenv("PHARE_ORG_API_KEY"); v == "" {
		t.Fatal("PHARE_ORG_API_KEY must be set for organization-scoped acceptance tests")
	} else {
		os.Setenv("PHARE_API_KEY", v)
	}
}

// TestAccProjectPreCheck validates that the project-scoped API key is set
// and copies it to PHARE_API_KEY for the provider to use
func TestAccProjectPreCheck(t *testing.T) {
	if v := os.Getenv("PHARE_PRJ_API_KEY"); v == "" {
		t.Fatal("PHARE_PRJ_API_KEY must be set for project-scoped acceptance tests")
	} else {
		os.Setenv("PHARE_API_KEY", v)
	}
}
