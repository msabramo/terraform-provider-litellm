package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func TestBuildMCPServerRequestIncludesSkipURLValidation(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:        types.StringValue("test-mcp"),
		URL:               types.StringValue("http://mcp.internal.svc.cluster.local:8000/mcp"),
		Transport:         types.StringValue("http"),
		SkipURLValidation: types.BoolValue(true),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	if got, ok := req["skip_url_validation"].(bool); !ok || !got {
		t.Fatalf("expected skip_url_validation=true, got %T: %v", req["skip_url_validation"], req["skip_url_validation"])
	}
}

func TestBuildMCPServerRequestOmitsSkipURLValidationWhenUnconfigured(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:        types.StringValue("test-mcp"),
		URL:               types.StringValue("https://example.com/mcp"),
		Transport:         types.StringValue("http"),
		SkipURLValidation: types.BoolNull(),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	if _, ok := req["skip_url_validation"]; ok {
		t.Fatalf("skip_url_validation should be omitted when unconfigured, got %v", req["skip_url_validation"])
	}
}

func TestBuildMCPServerRequestExtraHeadersList(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:   types.StringValue("test-mcp"),
		URL:          types.StringValue("https://example.com/mcp"),
		Transport:    types.StringValue("http"),
		ExtraHeaders: stringListValue("header-one", "header-two"),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	extraHeaders, ok := req["extra_headers"].([]string)
	if !ok {
		t.Fatalf("expected extra_headers to be []string, got %T: %v", req["extra_headers"], req["extra_headers"])
	}
	if len(extraHeaders) != 2 {
		t.Fatalf("expected 2 extra headers, got %d", len(extraHeaders))
	}
	if extraHeaders[0] != "header-one" || extraHeaders[1] != "header-two" {
		t.Fatalf("unexpected extra headers: %v", extraHeaders)
	}
}

func TestReadMCPServerExtraHeadersList(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_id":     "srv-extra-headers",
			"server_name":   "server-extra-headers",
			"url":           "https://example.com/mcp",
			"transport":     "http",
			"extra_headers": []interface{}{"header-one", "header-two"},
		})
	}))
	defer server.Close()

	r := &MCPServerResource{
		client: &Client{
			APIBase:    server.URL,
			APIKey:     "test-key",
			HTTPClient: server.Client(),
		},
	}

	data := MCPServerResourceModel{
		ID:           types.StringValue("srv-extra-headers"),
		ServerID:     types.StringValue("srv-extra-headers"),
		ExtraHeaders: types.ListUnknown(types.StringType),
	}

	if err := r.readMCPServer(context.Background(), &data); err != nil {
		t.Fatalf("readMCPServer returned error: %v", err)
	}

	if data.ExtraHeaders.IsUnknown() || data.ExtraHeaders.IsNull() {
		t.Fatal("extra_headers should be known and non-null after read")
	}

	var headers []string
	if diags := data.ExtraHeaders.ElementsAs(context.Background(), &headers, false); diags.HasError() {
		t.Fatalf("failed to decode extra_headers: %v", diags)
	}
	if len(headers) != 2 || headers[0] != "header-one" || headers[1] != "header-two" {
		t.Fatalf("unexpected extra_headers: %v", headers)
	}
}

func TestMCPServerUpgradeStateV0ToV1ExtraHeadersMapToList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	r := &MCPServerResource{}
	upgraders := r.UpgradeState(ctx)

	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("expected state upgrader for version 0")
	}

	v0State := map[string]interface{}{
		"id":          "srv-1",
		"server_id":   "srv-1",
		"server_name": "server-one",
		"url":         "https://example.com/mcp",
		"transport":   "http",
		"extra_headers": map[string]string{
			"header-two": "value-two",
			"header-one": "value-one",
		},
	}
	v0JSON, err := json.Marshal(v0State)
	if err != nil {
		t.Fatalf("failed to marshal v0 state: %v", err)
	}

	req := resource.UpgradeStateRequest{
		RawState: &tfprotov6.RawState{JSON: v0JSON},
	}
	resp := resource.UpgradeStateResponse{}

	upgrader.StateUpgrader(ctx, req, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
	}
	if resp.DynamicValue == nil {
		t.Fatal("expected DynamicValue to be set")
	}

	var upgraded map[string]interface{}
	if err := json.Unmarshal(resp.DynamicValue.JSON, &upgraded); err != nil {
		t.Fatalf("failed to unmarshal upgraded state: %v", err)
	}

	extraHeaders, ok := upgraded["extra_headers"].([]interface{})
	if !ok {
		t.Fatalf("expected extra_headers to be list after upgrade, got %T", upgraded["extra_headers"])
	}
	if len(extraHeaders) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(extraHeaders))
	}
	// Sorted for deterministic migration.
	if extraHeaders[0] != "header-one" || extraHeaders[1] != "header-two" {
		t.Fatalf("unexpected upgraded extra_headers: %v", extraHeaders)
	}
}

func TestBuildMCPServerRequestOAuthScopesInjectedIntoCredentials(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:  types.StringValue("test-mcp"),
		URL:         types.StringValue("https://example.com/mcp"),
		Transport:   types.StringValue("http"),
		Credentials: stringMapValue(map[string]string{"client_id": "abc"}),
		OAuthScopes: stringListValue("read", "write"),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	credentials, ok := req["credentials"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected credentials to be map[string]interface{}, got %T: %v", req["credentials"], req["credentials"])
	}
	if credentials["client_id"] != "abc" {
		t.Fatalf("expected credentials.client_id to be preserved, got %v", credentials["client_id"])
	}
	scopes, ok := credentials["scopes"].([]string)
	if !ok {
		t.Fatalf("expected credentials.scopes to be []string, got %T: %v", credentials["scopes"], credentials["scopes"])
	}
	if len(scopes) != 2 || scopes[0] != "read" || scopes[1] != "write" {
		t.Fatalf("unexpected credentials.scopes: %v", scopes)
	}
}

func TestBuildMCPServerRequestOAuthScopesWithoutCredentialsMap(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:  types.StringValue("test-mcp"),
		URL:         types.StringValue("https://example.com/mcp"),
		Transport:   types.StringValue("http"),
		OAuthScopes: stringListValue("read", "write"),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	credentials, ok := req["credentials"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected credentials to be map[string]interface{}, got %T: %v", req["credentials"], req["credentials"])
	}
	scopes, ok := credentials["scopes"].([]string)
	if !ok {
		t.Fatalf("expected credentials.scopes to be []string, got %T: %v", credentials["scopes"], credentials["scopes"])
	}
	if len(scopes) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(scopes))
	}
}

func TestBuildMCPServerRequestOmitsCredentialsWhenUnconfigured(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:  types.StringValue("test-mcp"),
		URL:         types.StringValue("https://example.com/mcp"),
		Transport:   types.StringValue("http"),
		OAuthScopes: types.ListNull(types.StringType),
		Credentials: types.MapNull(types.StringType),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	if _, ok := req["credentials"]; ok {
		t.Fatalf("credentials should be omitted when neither credentials nor oauth_scopes is set, got %v", req["credentials"])
	}
}

func TestBuildMCPServerRequestIncludesOAuthAndNetworkingFields(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:                types.StringValue("test-mcp"),
		URL:                       types.StringValue("https://example.com/mcp"),
		Transport:                 types.StringValue("http"),
		AvailableOnPublicInternet: types.BoolValue(false),
		OAuth2Flow:                types.StringValue("client_credentials"),
		Instructions:              types.StringValue("do the thing"),
		ToolNameToDisplayName:     stringMapValue(map[string]string{"search": "Search"}),
		ToolNameToDescription:     stringMapValue(map[string]string{"search": "Search the web"}),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	if got, ok := req["available_on_public_internet"].(bool); !ok || got {
		t.Fatalf("expected available_on_public_internet=false, got %T: %v", req["available_on_public_internet"], req["available_on_public_internet"])
	}
	if got, ok := req["oauth2_flow"].(string); !ok || got != "client_credentials" {
		t.Fatalf("expected oauth2_flow=client_credentials, got %v", req["oauth2_flow"])
	}
	if got, ok := req["instructions"].(string); !ok || got != "do the thing" {
		t.Fatalf("expected instructions, got %v", req["instructions"])
	}
	displayNames, ok := req["tool_name_to_display_name"].(map[string]string)
	if !ok || displayNames["search"] != "Search" {
		t.Fatalf("expected tool_name_to_display_name[search]=Search, got %T: %v", req["tool_name_to_display_name"], req["tool_name_to_display_name"])
	}
	descriptions, ok := req["tool_name_to_description"].(map[string]string)
	if !ok || descriptions["search"] != "Search the web" {
		t.Fatalf("expected tool_name_to_description[search]=Search the web, got %T: %v", req["tool_name_to_description"], req["tool_name_to_description"])
	}
}

func TestBuildMCPServerRequestOmitsOAuthAndNetworkingFieldsWhenUnconfigured(t *testing.T) {
	t.Parallel()

	r := &MCPServerResource{}
	data := &MCPServerResourceModel{
		ServerName:                types.StringValue("test-mcp"),
		URL:                       types.StringValue("https://example.com/mcp"),
		Transport:                 types.StringValue("http"),
		AvailableOnPublicInternet: types.BoolNull(),
		OAuth2Flow:                types.StringNull(),
		Instructions:              types.StringNull(),
		ToolNameToDisplayName:     types.MapNull(types.StringType),
		ToolNameToDescription:     types.MapNull(types.StringType),
	}

	req := r.buildMCPServerRequest(context.Background(), data)

	for _, key := range []string{"available_on_public_internet", "oauth2_flow", "instructions", "tool_name_to_display_name", "tool_name_to_description"} {
		if _, ok := req[key]; ok {
			t.Fatalf("%s should be omitted when unconfigured, got %v", key, req[key])
		}
	}
}

func TestReadMCPServerOAuthScopesFromCredentials(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_id":   "srv-scopes",
			"server_name": "server-scopes",
			"url":         "https://example.com/mcp",
			"transport":   "http",
			"credentials": map[string]interface{}{
				"scopes": []interface{}{"read", "write"},
			},
		})
	}))
	defer server.Close()

	r := &MCPServerResource{
		client: &Client{
			APIBase:    server.URL,
			APIKey:     "test-key",
			HTTPClient: server.Client(),
		},
	}

	data := MCPServerResourceModel{
		ID:          types.StringValue("srv-scopes"),
		ServerID:    types.StringValue("srv-scopes"),
		OAuthScopes: stringListValue("read", "write"),
	}

	if err := r.readMCPServer(context.Background(), &data); err != nil {
		t.Fatalf("readMCPServer returned error: %v", err)
	}

	var scopes []string
	if diags := data.OAuthScopes.ElementsAs(context.Background(), &scopes, false); diags.HasError() {
		t.Fatalf("failed to decode oauth_scopes: %v", diags)
	}
	if len(scopes) != 2 || scopes[0] != "read" || scopes[1] != "write" {
		t.Fatalf("unexpected oauth_scopes: %v", scopes)
	}
}

func TestReadMCPServerPreservesNullOAuthScopesWhenUnconfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_id":   "srv-scopes",
			"server_name": "server-scopes",
			"url":         "https://example.com/mcp",
			"transport":   "http",
			"credentials": map[string]interface{}{
				"scopes": []interface{}{"read", "write"},
			},
		})
	}))
	defer server.Close()

	r := &MCPServerResource{
		client: &Client{
			APIBase:    server.URL,
			APIKey:     "test-key",
			HTTPClient: server.Client(),
		},
	}

	data := MCPServerResourceModel{
		ID:          types.StringValue("srv-scopes"),
		ServerID:    types.StringValue("srv-scopes"),
		OAuthScopes: types.ListNull(types.StringType),
	}

	if err := r.readMCPServer(context.Background(), &data); err != nil {
		t.Fatalf("readMCPServer returned error: %v", err)
	}

	if !data.OAuthScopes.IsNull() {
		t.Fatalf("oauth_scopes should stay null when not set in config, got %v", data.OAuthScopes)
	}
}

func TestReadMCPServerReadsOAuthAndNetworkingFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_id":                    "srv-oauth",
			"server_name":                  "server-oauth",
			"url":                          "https://example.com/mcp",
			"transport":                    "http",
			"available_on_public_internet": false,
			"oauth2_flow":                  "client_credentials",
			"instructions":                 "do the thing",
			"tool_name_to_display_name": map[string]interface{}{
				"search": "Search",
			},
			"tool_name_to_description": map[string]interface{}{
				"search": "Search the web",
			},
		})
	}))
	defer server.Close()

	r := &MCPServerResource{
		client: &Client{
			APIBase:    server.URL,
			APIKey:     "test-key",
			HTTPClient: server.Client(),
		},
	}

	data := MCPServerResourceModel{
		ID:                        types.StringValue("srv-oauth"),
		ServerID:                  types.StringValue("srv-oauth"),
		AvailableOnPublicInternet: types.BoolValue(true),
		OAuth2Flow:                types.StringValue("authorization_code"),
		Instructions:              types.StringValue("placeholder"),
		ToolNameToDisplayName:     stringMapValue(map[string]string{"search": "old"}),
		ToolNameToDescription:     stringMapValue(map[string]string{"search": "old"}),
	}

	if err := r.readMCPServer(context.Background(), &data); err != nil {
		t.Fatalf("readMCPServer returned error: %v", err)
	}

	if data.AvailableOnPublicInternet.IsNull() || data.AvailableOnPublicInternet.ValueBool() {
		t.Fatalf("expected available_on_public_internet=false after read, got %v", data.AvailableOnPublicInternet)
	}
	if data.OAuth2Flow.ValueString() != "client_credentials" {
		t.Fatalf("expected oauth2_flow=client_credentials after read, got %v", data.OAuth2Flow)
	}
	if data.Instructions.ValueString() != "do the thing" {
		t.Fatalf("expected instructions to be read back, got %v", data.Instructions)
	}

	var displayNames map[string]string
	if diags := data.ToolNameToDisplayName.ElementsAs(context.Background(), &displayNames, false); diags.HasError() {
		t.Fatalf("failed to decode tool_name_to_display_name: %v", diags)
	}
	if displayNames["search"] != "Search" {
		t.Fatalf("expected tool_name_to_display_name[search]=Search, got %v", displayNames)
	}

	var descriptions map[string]string
	if diags := data.ToolNameToDescription.ElementsAs(context.Background(), &descriptions, false); diags.HasError() {
		t.Fatalf("failed to decode tool_name_to_description: %v", diags)
	}
	if descriptions["search"] != "Search the web" {
		t.Fatalf("expected tool_name_to_description[search]=Search the web, got %v", descriptions)
	}
}

func TestReadMCPServerPreservesNullOAuthFieldsWhenUnconfigured(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_id":                    "srv-oauth",
			"server_name":                  "server-oauth",
			"url":                          "https://example.com/mcp",
			"transport":                    "http",
			"available_on_public_internet": false,
			"oauth2_flow":                  "client_credentials",
			"instructions":                 "do the thing",
			"tool_name_to_display_name": map[string]interface{}{
				"search": "Search",
			},
			"tool_name_to_description": map[string]interface{}{
				"search": "Search the web",
			},
		})
	}))
	defer server.Close()

	r := &MCPServerResource{
		client: &Client{
			APIBase:    server.URL,
			APIKey:     "test-key",
			HTTPClient: server.Client(),
		},
	}

	data := MCPServerResourceModel{
		ID:                        types.StringValue("srv-oauth"),
		ServerID:                  types.StringValue("srv-oauth"),
		AvailableOnPublicInternet: types.BoolNull(),
		OAuth2Flow:                types.StringNull(),
		Instructions:              types.StringNull(),
		ToolNameToDisplayName:     types.MapNull(types.StringType),
		ToolNameToDescription:     types.MapNull(types.StringType),
	}

	if err := r.readMCPServer(context.Background(), &data); err != nil {
		t.Fatalf("readMCPServer returned error: %v", err)
	}

	if !data.AvailableOnPublicInternet.IsNull() {
		t.Fatalf("available_on_public_internet should stay null, got %v", data.AvailableOnPublicInternet)
	}
	if !data.OAuth2Flow.IsNull() {
		t.Fatalf("oauth2_flow should stay null, got %v", data.OAuth2Flow)
	}
	if !data.Instructions.IsNull() {
		t.Fatalf("instructions should stay null, got %v", data.Instructions)
	}
	if !data.ToolNameToDisplayName.IsNull() {
		t.Fatalf("tool_name_to_display_name should stay null, got %v", data.ToolNameToDisplayName)
	}
	if !data.ToolNameToDescription.IsNull() {
		t.Fatalf("tool_name_to_description should stay null, got %v", data.ToolNameToDescription)
	}
}

func TestReadMCPServerResolvesUnknownNestedToolCostMap(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_id":   "srv-1",
			"server_name": "server-one",
			"url":         "https://example.com/mcp",
			"transport":   "http",
			"mcp_info": map[string]interface{}{
				"mcp_server_cost_info": map[string]interface{}{},
			},
		})
	}))
	defer server.Close()

	r := &MCPServerResource{
		client: &Client{
			APIBase:    server.URL,
			APIKey:     "test-key",
			HTTPClient: server.Client(),
		},
	}

	data := MCPServerResourceModel{
		ID:       types.StringValue("srv-1"),
		ServerID: types.StringValue("srv-1"),
		MCPInfo: &MCPInfoModel{
			MCPServerCostInfo: &MCPServerCostInfoModel{
				ToolNameToCostPerQuery: types.MapUnknown(types.Float64Type),
			},
		},
	}

	if err := r.readMCPServer(context.Background(), &data); err != nil {
		t.Fatalf("readMCPServer returned error: %v", err)
	}

	if data.MCPInfo == nil || data.MCPInfo.MCPServerCostInfo == nil {
		t.Fatal("mcp_info.mcp_server_cost_info should be present after read")
	}
	if data.MCPInfo.MCPServerCostInfo.ToolNameToCostPerQuery.IsUnknown() {
		t.Fatal("tool_name_to_cost_per_query should be known after read")
	}
}

func TestReadMCPServerReadsNestedToolCostMap(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"server_id":   "srv-2",
			"server_name": "server-two",
			"url":         "https://example.com/mcp",
			"transport":   "http",
			"mcp_info": map[string]interface{}{
				"mcp_server_cost_info": map[string]interface{}{
					"tool_name_to_cost_per_query": map[string]interface{}{
						"search": 0.25,
					},
				},
			},
		})
	}))
	defer server.Close()

	r := &MCPServerResource{
		client: &Client{
			APIBase:    server.URL,
			APIKey:     "test-key",
			HTTPClient: server.Client(),
		},
	}

	data := MCPServerResourceModel{
		ID:       types.StringValue("srv-2"),
		ServerID: types.StringValue("srv-2"),
		MCPInfo: &MCPInfoModel{
			MCPServerCostInfo: &MCPServerCostInfoModel{
				ToolNameToCostPerQuery: types.MapUnknown(types.Float64Type),
			},
		},
	}

	if err := r.readMCPServer(context.Background(), &data); err != nil {
		t.Fatalf("readMCPServer returned error: %v", err)
	}

	toolCosts := map[string]float64{}
	if diags := data.MCPInfo.MCPServerCostInfo.ToolNameToCostPerQuery.ElementsAs(context.Background(), &toolCosts, false); diags.HasError() {
		t.Fatalf("failed to decode tool_name_to_cost_per_query: %v", diags)
	}
	if got := toolCosts["search"]; got != 0.25 {
		t.Fatalf("expected search cost 0.25, got %v", got)
	}
}
