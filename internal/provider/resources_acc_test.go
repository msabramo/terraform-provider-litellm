package provider

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Acceptance tests that drive a real create/read cycle against the LiteLLM
// backend using the project's own smoke configs in internal_testing/resources/
// as the single source of truth (loaded via TestStep.ConfigFile). This keeps
// the acceptance tests and the manually-run smoke configs from drifting apart:
// the same HCL the maintainer runs by hand is exercised here with coverage.
//
// TestAccSmokeConfigs auto-discovers *_minimal.tf files — drop a new one in and
// it is picked up automatically, no test code change. A config is run standalone
// only if it is self-contained: every litellm_* resource it references is also
// defined in the same file, and it doesn't depend on an externally pre-existing
// model. Configs that compose multiple files (team_member -> litellm_team, ...)
// or need a pre-existing model (access_group) are skipped with a logged reason;
// those remain part of the manual, multi-file smoke flow.
//
// Gated by TF_ACC; see acceptance_test.go. The framework injects the provider
// configuration, so the config files need no provider block.

const smokeResourcesDir = "../../internal_testing/resources"

var (
	// resource "litellm_x" "y" { -> address litellm_x.y
	reResourceDef = regexp.MustCompile(`resource\s+"(litellm_[a-z_]+)"\s+"([a-zA-Z0-9_-]+)"`)
	// a reference to another resource's attribute: litellm_x.y.attr
	reResourceRef = regexp.MustCompile(`(litellm_[a-z_]+\.[a-zA-Z0-9_-]+)\.`)
	// model_names = ["gpt-4o-mini"] and similar: needs a model that must pre-exist
	reNeedsModel = regexp.MustCompile(`model_names\s*=\s*\[\s*"gpt`)
)

// smokeCase is a discovered, runnable smoke config.
type smokeCase struct {
	file      string   // absolute path
	name      string   // subtest name (file base without _minimal.tf)
	addresses []string // resource addresses defined in the file
}

// discoverSelfContainedSmokeConfigs scans smokeResourcesDir for *_minimal.tf
// files that can be applied on their own, returning the runnable cases and a
// map of file base -> skip reason for the rest.
func discoverSelfContainedSmokeConfigs(t *testing.T) ([]smokeCase, map[string]string) {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(smokeResourcesDir, "*_minimal.tf"))
	if err != nil {
		t.Fatalf("globbing smoke configs: %v", err)
	}

	var runnable []smokeCase
	skipped := map[string]string{}

	for _, path := range matches {
		base := filepath.Base(path)
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", base, err)
		}
		content := string(src)

		defined := map[string]bool{}
		var addresses []string
		for _, m := range reResourceDef.FindAllStringSubmatch(content, -1) {
			addr := m[1] + "." + m[2]
			if !defined[addr] {
				defined[addr] = true
				addresses = append(addresses, addr)
			}
		}
		if len(addresses) == 0 {
			skipped[base] = "no litellm resource defined"
			continue
		}

		if reNeedsModel.MatchString(content) {
			skipped[base] = "references a pre-existing model (needs a model created first)"
			continue
		}

		// Self-contained iff every referenced address is defined in this file.
		selfContained := true
		for _, m := range reResourceRef.FindAllStringSubmatch(content, -1) {
			if !defined[m[1]] {
				selfContained = false
				break
			}
		}
		if !selfContained {
			skipped[base] = "references resources defined in other smoke files"
			continue
		}

		name := base
		name = name[:len(name)-len("_minimal.tf")]
		runnable = append(runnable, smokeCase{file: path, name: name, addresses: addresses})
	}

	return runnable, skipped
}

// TestAccSmokeConfigs runs every self-contained smoke config through a real
// apply/read/destroy cycle and asserts each resource it defines is created.
func TestAccSmokeConfigs(t *testing.T) {
	runnable, skipped := discoverSelfContainedSmokeConfigs(t)

	if len(runnable) == 0 {
		t.Fatal("no self-contained smoke configs discovered")
	}
	for file, reason := range skipped {
		t.Logf("skipping %s: %s", file, reason)
	}

	for _, tc := range runnable {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			checks := make([]resource.TestCheckFunc, 0, len(tc.addresses))
			for _, addr := range tc.addresses {
				checks = append(checks, resource.TestCheckResourceAttrSet(addr, "id"))
			}
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{{
					ConfigFile: config.StaticFile(tc.file),
					Check:      resource.ComposeAggregateTestCheckFunc(checks...),
				}},
			})
		})
	}
}
