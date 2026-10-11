package testacc

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"terraform-provider-phare/internal/client"
)

// SweepMaxAge is the age after which a resource created by the acceptance
// tests is considered leaked. It must stay above the CI job timeout so that
// a sweep never deletes the resources of a concurrent test run.
const SweepMaxAge = time.Hour

// TestUserEmail is the email of the user added as member of test projects
const TestUserEmail = "terraform@phare.io"

const sweepPageSize = 100

// CreateRunProject creates the project holding every resource created by an
// organization-scoped test run. Deleting it deletes all of its resources.
func CreateRunProject(ctx context.Context, c *client.Client) (*client.ProjectResponse, error) {
	users, err := c.ListUsers(ctx, TestUserEmail, 1, sweepPageSize)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	members := make([]int64, len(users))
	for i, user := range users {
		members[i] = user.ID
	}

	return c.CreateProject(ctx, &client.ProjectRequest{
		Name:    RandomName("run"),
		Members: members,
	})
}

// SweepProjects deletes the projects leaked by previous organization-scoped
// test runs, which also deletes all of their resources
func SweepProjects(ctx context.Context, c *client.Client) error {
	projects, err := listAll(func(page int) ([]*client.ProjectResponse, error) {
		return c.ListProjects(ctx, page, sweepPageSize)
	})
	if err != nil {
		return fmt.Errorf("listing projects: %w", err)
	}

	for _, project := range projects {
		if !isLeaked(project.Name, project.CreatedAt) {
			continue
		}

		log.Printf("[INFO] Sweeping leaked project %q (%d)", project.Name, project.ID)
		if err := c.DeleteProject(ctx, project.ID); err != nil && !client.IsNotFoundError(err) {
			return fmt.Errorf("deleting project %d: %w", project.ID, err)
		}
	}

	return nil
}

// SweepProjectResources deletes the resources leaked by previous
// project-scoped test runs in the project of the client's API key
func SweepProjectResources(ctx context.Context, c *client.Client) error {
	// Status pages first, as they reference monitors
	statusPages, err := listAll(func(page int) ([]*client.StatusPageResponse, error) {
		return c.ListStatusPages(ctx, page, sweepPageSize, nil)
	})
	if err != nil {
		return fmt.Errorf("listing status pages: %w", err)
	}

	for _, statusPage := range statusPages {
		if !isLeaked(statusPage.Name, statusPage.CreatedAt) {
			continue
		}

		log.Printf("[INFO] Sweeping leaked status page %q (%d)", statusPage.Name, statusPage.ID)
		if err := c.DeleteStatusPage(ctx, statusPage.ID); err != nil && !client.IsNotFoundError(err) {
			return fmt.Errorf("deleting status page %d: %w", statusPage.ID, err)
		}
	}

	monitors, err := listAll(func(page int) ([]*client.MonitorResponse, error) {
		return c.ListMonitors(ctx, page, sweepPageSize, nil)
	})
	if err != nil {
		return fmt.Errorf("listing monitors: %w", err)
	}

	for _, monitor := range monitors {
		if !isLeaked(monitor.Name, monitor.CreatedAt) {
			continue
		}

		log.Printf("[INFO] Sweeping leaked monitor %q (%d)", monitor.Name, monitor.ID)
		if err := c.DeleteMonitor(ctx, monitor.ID); err != nil && !client.IsNotFoundError(err) {
			return fmt.Errorf("deleting monitor %d: %w", monitor.ID, err)
		}
	}

	// Alert rules have no name, the project of the API key is expected to be
	// dedicated to the acceptance tests
	alertRules, err := listAll(func(page int) ([]*client.AlertRuleResponse, error) {
		return c.ListAlertRules(ctx, page, sweepPageSize)
	})
	if err != nil {
		return fmt.Errorf("listing alert rules: %w", err)
	}

	for _, alertRule := range alertRules {
		if alertRule.ProjectID == nil || !isStale(alertRule.CreatedAt) {
			continue
		}

		log.Printf("[INFO] Sweeping leaked alert rule %d", alertRule.ID)
		if err := c.DeleteAlertRule(ctx, alertRule.ID); err != nil && !client.IsNotFoundError(err) {
			return fmt.Errorf("deleting alert rule %d: %w", alertRule.ID, err)
		}
	}

	return nil
}

// isLeaked reports whether a resource was created by the acceptance tests
// more than SweepMaxAge ago
func isLeaked(name, createdAt string) bool {
	return strings.HasPrefix(name, ResourcePrefix) && isStale(createdAt)
}

// isStale reports whether a resource was created more than SweepMaxAge ago.
// Resources with an unparsable creation date are never considered stale.
func isStale(createdAt string) bool {
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return false
	}

	return time.Since(created) > SweepMaxAge
}

// listAll collects every page of a paginated list endpoint
func listAll[T any](list func(page int) ([]T, error)) ([]T, error) {
	var all []T

	for page := 1; ; page++ {
		items, err := list(page)
		if err != nil {
			return nil, err
		}

		all = append(all, items...)

		if len(items) < sweepPageSize {
			return all, nil
		}
	}
}
