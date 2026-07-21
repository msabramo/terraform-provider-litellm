package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccBudgetResource is an example acceptance test: it applies a
// litellm_budget, verifies the created attributes in state, then verifies the
// resource imports cleanly. It runs a real plan/apply/import/destroy cycle
// against the LiteLLM backend and only executes when TF_ACC is set.
func TestAccBudgetResource(t *testing.T) {
	budgetID := "tf-acc-budget"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read.
			{
				Config: testAccBudgetConfig(budgetID, 100.0, 60),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("litellm_budget.test", "budget_id", budgetID),
					resource.TestCheckResourceAttr("litellm_budget.test", "max_budget", "100"),
					resource.TestCheckResourceAttr("litellm_budget.test", "rpm_limit", "60"),
					resource.TestCheckResourceAttrSet("litellm_budget.test", "id"),
				),
			},
			// Update in place.
			{
				Config: testAccBudgetConfig(budgetID, 250.0, 120),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("litellm_budget.test", "max_budget", "250"),
					resource.TestCheckResourceAttr("litellm_budget.test", "rpm_limit", "120"),
				),
			},
			// Import.
			{
				ResourceName:      "litellm_budget.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccBudgetConfig(budgetID string, maxBudget float64, rpmLimit int) string {
	return fmt.Sprintf(`
provider "litellm" {}

resource "litellm_budget" "test" {
  budget_id  = %q
  max_budget = %g
  rpm_limit  = %d
}
`, budgetID, maxBudget, rpmLimit)
}
