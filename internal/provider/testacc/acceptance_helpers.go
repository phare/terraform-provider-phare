package testacc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider"
)

// ResourcePrefix is prepended to the name of every resource created by the
// acceptance tests, so leaked resources can be identified and swept
const ResourcePrefix = "tfacc-"

// TestAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"phare": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// randomCharSet is the character set of random name suffixes, valid in
// names, status page subdomains and tags
const randomCharSet = "abcdefghijklmnopqrstuvwxyz0123456789"

// RandomName returns a unique resource name, so that concurrent test runs
// against the same account never share resources. Names are kept short as
// project names are limited to 25 characters and status page names to 30.
func RandomName(name string) string {
	return ResourcePrefix + name + "-" + acctest.RandStringFromCharSet(8, randomCharSet)
}

// RandomSubdomain returns a unique status page subdomain. Subdomains are
// unique across the whole platform and limited to 30 characters.
func RandomSubdomain() string {
	return ResourcePrefix + acctest.RandStringFromCharSet(10, randomCharSet)
}

// RandomTag returns a unique tag, so that tag filters only match resources
// created by the current test run
func RandomTag() string {
	return "tfacc:" + acctest.RandStringFromCharSet(10, randomCharSet)
}

// TestDataFile returns the absolute path to a file of the testdata directory,
// formatted for use in a Terraform configuration
func TestDataFile(name string) string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.ToSlash(filepath.Join(filepath.Dir(currentFile), "testdata", name))
}

// StatusPageThemeBlock is a reusable theme configuration for status page tests
const StatusPageThemeBlock = `
  theme {
    rounded      = true
    border_width = 2

    light {
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      empty                = "#d3d3d3"
      background           = "#ffffff"
      foreground           = "#000000"
      foreground_muted     = "#737373"
      background_card      = "#fafafa"
    }

    dark {
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      empty                = "#d3d3d3"
      background           = "#111111"
      foreground           = "#ffffff"
      foreground_muted     = "#959595"
      background_card      = "#1a1a1a"
    }
  }
`

// NewTestClient creates an API client for the given API key, scoped to the
// given project when projectSlug is set
func NewTestClient(apiKey, projectSlug string) (*client.Client, error) {
	baseURL := "https://api.phare.io"
	if v := os.Getenv("PHARE_BASE_URL"); v != "" {
		baseURL = v
	}

	isProjectScoped := !strings.HasPrefix(apiKey, "pha_org_")

	return client.NewClient(baseURL, apiKey, 0, "", projectSlug, "test", "", isProjectScoped)
}

// importStateIdWithProjectScope returns an ImportStateIdFunc building the
// "<project_scope>/<id>" import ID required by organization-scoped API keys
func importStateIdWithProjectScope(resourceName, projectScope string) testingresource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", resourceName)
		}

		return projectScope + "/" + rs.Primary.ID, nil
	}
}

// CheckDestroy verifies that every Phare resource of the state has been
// deleted from the API, using the API key the provider was configured with
// and the given project scope (empty for project-scoped API keys)
func CheckDestroy(projectScope string) testingresource.TestCheckFunc {
	return func(s *terraform.State) error {
		apiKey := os.Getenv("PHARE_API_KEY")

		scopedClient, err := NewTestClient(apiKey, projectScope)
		if err != nil {
			return err
		}

		unscopedClient, err := NewTestClient(apiKey, "")
		if err != nil {
			return err
		}

		ctx := context.Background()

		for name, rs := range s.RootModule().Resources {
			// Data sources read existing objects that are not destroyed
			if strings.HasPrefix(name, "data.") {
				continue
			}

			id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
			if err != nil {
				continue
			}

			switch rs.Type {
			case "phare_project":
				_, err = unscopedClient.GetProject(ctx, id)
			case "phare_alert_rule":
				_, err = scopedClient.GetAlertRule(ctx, id)
			case "phare_uptime_monitor_http", "phare_uptime_monitor_tcp", "phare_uptime_monitor_icmp":
				_, err = scopedClient.GetMonitor(ctx, id)
			case "phare_uptime_status_page":
				_, err = scopedClient.GetStatusPage(ctx, id)
			default:
				continue
			}

			if err == nil {
				return fmt.Errorf("%s %d still exists", rs.Type, id)
			}
			if !client.IsNotFoundError(err) {
				return fmt.Errorf("checking whether %s %d was destroyed: %w", rs.Type, id, err)
			}
		}

		return nil
	}
}
