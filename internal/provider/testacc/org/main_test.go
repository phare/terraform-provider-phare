package testacc_org

import (
	"context"
	"log"
	"os"
	"testing"

	"terraform-provider-phare/internal/provider/testacc"
)

// runProjectSlug is the slug of the project created for the current test run.
// Every project-scoped resource is created in it, so deleting it at the end
// of the run cleans up anything a failed test left behind.
var runProjectSlug string

func TestMain(m *testing.M) {
	if os.Getenv("TF_ACC") == "" || os.Getenv("PHARE_ORG_API_KEY") == "" {
		os.Exit(m.Run())
	}

	os.Exit(runWithProject(m))
}

// runWithProject sweeps the projects leaked by previous runs, creates the
// project of the current run, runs the tests and deletes the project
func runWithProject(m *testing.M) int {
	ctx := context.Background()

	c, err := testacc.NewTestClient(os.Getenv("PHARE_ORG_API_KEY"), "")
	if err != nil {
		log.Printf("[ERROR] Creating API client: %s", err)
		return 1
	}

	if err := testacc.SweepProjects(ctx, c); err != nil {
		log.Printf("[WARN] Sweeping leaked projects: %s", err)
	}

	if err := testacc.SweepIntegrations(ctx, c); err != nil {
		log.Printf("[WARN] Sweeping leaked integrations: %s", err)
	}

	project, err := testacc.CreateRunProject(ctx, c)
	if err != nil {
		log.Printf("[ERROR] Creating test run project: %s", err)
		return 1
	}

	runProjectSlug = project.Slug

	defer func() {
		if err := c.DeleteProject(ctx, project.ID); err != nil {
			log.Printf("[ERROR] Deleting test run project %q (%d): %s", project.Slug, project.ID, err)
		}
	}()

	return m.Run()
}
