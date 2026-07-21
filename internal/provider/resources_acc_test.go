package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests that drive a real create/read cycle against the LiteLLM
// backend using the project's own smoke configs in internal_testing/resources/
// as the single source of truth (loaded via TestStep.ConfigFile). This keeps
// the acceptance tests and the manual smoke configs from drifting apart: the
// same HCL the maintainer runs by hand is exercised here with coverage.
//
// Only self-contained minimal configs are loaded directly. Configs that
// reference other resources (team_member, organization_member, ...) are left to
// the manual smoke flow, which composes multiple files into one state.
//
// Gated by TF_ACC; see acceptance_test.go. The framework injects the provider
// configuration from testAccProtoV6ProviderFactories, so the config files need
// no provider block (and the smoke files don't have one).

const smokeResourcesDir = "../../internal_testing/resources/"

// smokeResourceCases lists the self-contained minimal smoke configs and the
// resource address each defines, so we can assert it lands in state.
var smokeResourceCases = []struct {
	name    string // subtest name
	file    string // file under internal_testing/resources/
	address string // resource address defined in that file
}{
	// budget is covered in depth (create/update/import) by TestAccBudgetResource.
	{"tag", "tag_minimal.tf", "litellm_tag.minimal"},
	{"model", "model_minimal.tf", "litellm_model.minimal"},
	{"user", "user_minimal.tf", "litellm_user.minimal"},
	{"organization", "organization_minimal.tf", "litellm_organization.minimal"},
	{"team", "team_minimal.tf", "litellm_team.minimal"},
	{"key", "key_minimal.tf", "litellm_key.minimal"},
	{"credential", "credential_minimal.tf", "litellm_credential.minimal"},
	{"guardrail", "guardrail_minimal.tf", "litellm_guardrail.minimal"},
	{"search_tool", "search_tool_minimal.tf", "litellm_search_tool.minimal"},
	{"vector_store", "vector_store_minimal.tf", "litellm_vector_store.minimal"},
	{"prompt", "prompt_minimal.tf", "litellm_prompt.minimal"},
	{"mcp_server", "mcp_server_minimal.tf", "litellm_mcp_server.minimal"},
	{"fallback", "fallback_minimal.tf", "litellm_fallback.minimal"},
}

// TestAccSmokeConfigs runs each self-contained smoke config through a real
// apply/read/destroy cycle and asserts the resource it defines is created.
func TestAccSmokeConfigs(t *testing.T) {
	for _, tc := range smokeResourceCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					ConfigFile: config.StaticFile(smokeResourcesDir + tc.file),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet(tc.address, "id"),
					),
				}},
			})
		})
	}
}
