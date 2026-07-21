# litellm_model - additional_model_info
#
# Smoke test for the additional_model_info map added in PR #143 (issue #140):
# arbitrary model_info fields (primarily capability flags for models missing
# from the LiteLLM cost map) are sent on create and read back for the keys the
# user configured.
#
# Run with:  make smoke resources=model_additional_info.tf

resource "litellm_model" "additional_info" {
  model_name          = "test-model-additional-info"
  custom_llm_provider = "openai"
  base_model          = "gpt-4o-mini"
  mode                = "chat"

  additional_model_info = {
    supports_vision           = "true"
    supports_function_calling = "true"
  }
}

output "model_additional_info_id" {
  value = litellm_model.additional_info.id
}

output "model_additional_info_flags" {
  value = litellm_model.additional_info.additional_model_info
}
