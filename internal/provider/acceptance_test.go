package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// Acceptance tests exercise the provider end-to-end against a real LiteLLM
// backend by running actual terraform plan/apply/import/destroy through the
// terraform-plugin-testing framework. They only run when TF_ACC is set.
//
// A disposable backend is provided by internal_testing/docker-compose.yml:
//
//	make local            # start LiteLLM + Postgres on localhost:4000
//	make testacc          # TF_ACC=1 go test ./internal/provider/ -run TestAcc
//	cd internal_testing && docker compose down -v   # throw it away
//
// Configuration comes from environment variables (with defaults matching the
// bundled compose file), so no credentials are hard-coded:
//
//	LITELLM_API_BASE  (default http://localhost:4000)
//	LITELLM_API_KEY   (default sk-testing-key)

const (
	defaultAccAPIBase = "http://localhost:4000"
	defaultAccAPIKey  = "sk-testing-key"
)

// testAccProtoV6ProviderFactories wires the provider under test into the
// terraform-plugin-testing harness via the protocol-6 server.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"litellm": providerserver.NewProtocol6WithError(New("test")()),
}

// testAccPreCheck validates that the acceptance-test prerequisites are present
// and normalizes the backend env vars to their compose defaults when unset. It
// must be called from every acceptance test's PreCheck.
func testAccPreCheck(t *testing.T) {
	t.Helper()

	if os.Getenv("LITELLM_API_BASE") == "" {
		t.Setenv("LITELLM_API_BASE", defaultAccAPIBase)
	}
	if os.Getenv("LITELLM_API_KEY") == "" {
		t.Setenv("LITELLM_API_KEY", defaultAccAPIKey)
	}
}
