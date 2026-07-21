# litellm_key - Router settings
#
# Smoke test for the typed router_settings block added in PR #98
# (https://github.com/ncecere/terraform-provider-litellm/pull/98).
#
# Verifies that a per-key router_settings block applies, reads back, and does
# not produce a perpetual diff. router_settings is a SingleNestedAttribute, so
# it uses attribute-assignment syntax (router_settings = { ... }), not a nested
# block.
#
# Run with:  make smoke resources=key_router_settings.tf

resource "litellm_key" "router_settings" {
  key_alias = "test-key-router-settings"

  router_settings = {
    num_retries   = 3
    timeout       = 30.0
    allowed_fails = 2
    cooldown_time = 60.0

    fallbacks = [
      {
        model           = "gpt-4o"
        fallback_models = ["gpt-4o-mini"]
      }
    ]

    retry_policy = {
      rate_limit_error_retries      = 3
      timeout_error_retries         = 2
      internal_server_error_retries = 1
    }
  }
}

output "key_router_settings_id" {
  value = litellm_key.router_settings.id
}

output "key_router_settings_num_retries" {
  value = litellm_key.router_settings.router_settings.num_retries
}
