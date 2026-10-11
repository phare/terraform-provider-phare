package testacc_project

import (
	"context"
	"log"
	"os"
	"testing"

	"terraform-provider-phare/internal/provider/testacc"
)

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") != "" && os.Getenv("PHARE_PRJ_API_KEY") != "" {
		sweep()
	}

	os.Exit(m.Run())
}

// sweep deletes the resources leaked by previous runs. The project-scoped API
// key cannot create projects, so resources are matched by name and age.
func sweep() {
	c, err := testacc.NewTestClient(os.Getenv("PHARE_PRJ_API_KEY"), "")
	if err != nil {
		log.Printf("[WARN] Creating API client: %s", err)
		return
	}

	if err := testacc.SweepProjectResources(context.Background(), c); err != nil {
		log.Printf("[WARN] Sweeping leaked resources: %s", err)
	}

	// After the alert rules, as integrations used by alert rules cannot be deleted
	if err := testacc.SweepIntegrations(context.Background(), c); err != nil {
		log.Printf("[WARN] Sweeping leaked integrations: %s", err)
	}
}
