package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests for standalone resources (no cross-resource dependencies),
// each exercising a real create -> read -> destroy cycle against the LiteLLM
// backend. Configs mirror the minimal smoke configs in internal_testing/.
// Gated by TF_ACC; see acceptance_test.go.

func TestAccTagResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_tag" "test" {
  name        = "tf-acc-tag"
  description = "acc test tag"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_tag.test", "name", "tf-acc-tag"),
				resource.TestCheckResourceAttrSet("litellm_tag.test", "id"),
			),
		}},
	})
}

func TestAccAccessGroupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			// access_group references model_names that must already exist, so
			// create a model first and point the group at it.
			Config: `
provider "litellm" {}
resource "litellm_model" "for_ag" {
  model_name          = "tf-acc-ag-model"
  custom_llm_provider = "openai"
  base_model          = "gpt-4o-mini"
}
resource "litellm_access_group" "test" {
  access_group = "tf-acc-access-group"
  model_names  = [litellm_model.for_ag.model_name]
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_access_group.test", "access_group", "tf-acc-access-group"),
				resource.TestCheckResourceAttrSet("litellm_access_group.test", "id"),
			),
		}},
	})
}

func TestAccModelResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_model" "test" {
  model_name          = "tf-acc-model"
  custom_llm_provider = "openai"
  base_model          = "gpt-4o-mini"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_model.test", "model_name", "tf-acc-model"),
				resource.TestCheckResourceAttrSet("litellm_model.test", "id"),
			),
		}},
	})
}

func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_user" "test" {
  user_alias      = "tf-acc-user"
  auto_create_key = false
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_user.test", "user_alias", "tf-acc-user"),
				resource.TestCheckResourceAttrSet("litellm_user.test", "id"),
			),
		}},
	})
}

func TestAccOrganizationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_organization" "test" {
  organization_alias = "tf-acc-org"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_organization.test", "organization_alias", "tf-acc-org"),
				resource.TestCheckResourceAttrSet("litellm_organization.test", "id"),
			),
		}},
	})
}

func TestAccTeamResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_team" "test" {
  team_alias = "tf-acc-team"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_team.test", "team_alias", "tf-acc-team"),
				resource.TestCheckResourceAttrSet("litellm_team.test", "id"),
			),
		}},
	})
}

func TestAccKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_key" "test" {
  key_alias = "tf-acc-key"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_key.test", "key_alias", "tf-acc-key"),
				resource.TestCheckResourceAttrSet("litellm_key.test", "id"),
			),
		}},
	})
}

func TestAccCredentialResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_credential" "test" {
  credential_name = "tf-acc-cred"
  credential_info = {
    description = "acc test credential"
  }
  credential_values = {
    api_key = "sk-fake-acc-key"
  }
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_credential.test", "credential_name", "tf-acc-cred"),
				resource.TestCheckResourceAttrSet("litellm_credential.test", "id"),
			),
		}},
	})
}

func TestAccGuardrailResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_guardrail" "test" {
  guardrail_name = "tf-acc-guardrail"
  guardrail      = "aporia"
  mode           = "pre_call"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_guardrail.test", "guardrail_name", "tf-acc-guardrail"),
				resource.TestCheckResourceAttrSet("litellm_guardrail.test", "id"),
			),
		}},
	})
}

func TestAccSearchToolResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_search_tool" "test" {
  search_tool_name = "tf-acc-search"
  search_provider  = "tavily"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_search_tool.test", "search_tool_name", "tf-acc-search"),
				resource.TestCheckResourceAttrSet("litellm_search_tool.test", "id"),
			),
		}},
	})
}

func TestAccVectorStoreResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_vector_store" "test" {
  vector_store_name   = "tf-acc-vs"
  custom_llm_provider = "openai"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_vector_store.test", "vector_store_name", "tf-acc-vs"),
				resource.TestCheckResourceAttrSet("litellm_vector_store.test", "id"),
			),
		}},
	})
}

func TestAccPromptResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_prompt" "test" {
  prompt_id          = "tf-acc-prompt"
  prompt_integration = "dotprompt"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_prompt.test", "prompt_id", "tf-acc-prompt"),
				resource.TestCheckResourceAttrSet("litellm_prompt.test", "id"),
			),
		}},
	})
}

func TestAccMCPServerResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "litellm" {}
resource "litellm_mcp_server" "test" {
  server_name = "tf_acc_mcp"
  url         = "https://example.com/mcp"
  transport   = "sse"
}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("litellm_mcp_server.test", "server_name", "tf_acc_mcp"),
				resource.TestCheckResourceAttrSet("litellm_mcp_server.test", "id"),
			),
		}},
	})
}
