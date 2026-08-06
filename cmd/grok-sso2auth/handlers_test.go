package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestPluginRegisterEnvelope(t *testing.T) {
	raw, err := handleMethod(pluginabi.MethodPluginRegister, nil)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	if !env.OK {
		t.Fatalf("not ok: %s", raw)
	}
	var reg registration
	if err := json.Unmarshal(env.Result, &reg); err != nil {
		t.Fatal(err)
	}
	if reg.SchemaVersion == 0 {
		t.Fatal("schema_version missing")
	}
	if !reg.Capabilities.ManagementAPI || !reg.Capabilities.CommandLinePlugin {
		t.Fatalf("capabilities = %+v", reg.Capabilities)
	}
	if reg.Metadata.Name == "" || reg.Metadata.Version == "" {
		t.Fatalf("metadata = %+v", reg.Metadata)
	}
}

func TestManagementRegisterRoutes(t *testing.T) {
	raw, err := handleMethod(pluginabi.MethodManagementRegister, nil)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	var reg managementRegistration
	if err := json.Unmarshal(env.Result, &reg); err != nil {
		t.Fatal(err)
	}
	if len(reg.Resources) == 0 {
		t.Fatal("expected resource")
	}
	if reg.Resources[0].Path != "/status" {
		t.Fatalf("resource path = %q, want /status", reg.Resources[0].Path)
	}
	if !strings.Contains(string(env.Result), "\"resources\"") {
		t.Fatalf("management registration should use documented lowercase resources field: %s", env.Result)
	}
	if len(reg.Routes) < 5 {
		t.Fatalf("routes = %+v", reg.Routes)
	}
	foundRuntime := false
	for _, route := range reg.Routes {
		if route.Path == mgmtAuthRuntimePath {
			foundRuntime = true
			break
		}
	}
	if !foundRuntime {
		t.Fatalf("runtime route missing: %+v", reg.Routes)
	}
}

func TestManagementUIServesHTML(t *testing.T) {
	req, _ := json.Marshal(managementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/grok-sso2auth/status",
	})
	raw, err := handleMethod(pluginabi.MethodManagementHandle, req)
	if err != nil {
		t.Fatal(err)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatal(err)
	}
	var resp managementResponse
	if err := json.Unmarshal(env.Result, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	body := string(resp.Body)
	if !strings.Contains(body, "Grok SSO") {
		t.Fatalf("body missing title, len=%d", len(body))
	}
	if !strings.Contains(body, "host.auth.get_runtime") {
		t.Fatalf("body missing runtime callback documentation, len=%d", len(body))
	}
	ct := resp.Headers.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type=%q", ct)
	}
}

func TestConvertRequiresSSO(t *testing.T) {
	status, payload := handleConvert([]byte(`{}`), false)
	if status == http.StatusOK {
		t.Fatalf("expected error status, got %d %#v", status, payload)
	}
}

func TestImportRequiresXAI(t *testing.T) {
	status, _ := handleImport([]byte(`{"json":{"type":"gemini","email":"demo@example.com"}}`))
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", status, http.StatusBadRequest)
	}
}

func TestSummarizeAuthOmitsPhysicalPath(t *testing.T) {
	summary := summarizeAuth(pluginapi.HostAuthFileEntry{
		AuthIndex: "auth-1",
		Name:      "xai-demo.json",
		Type:      "xai",
		Email:     "demo@example.com",
		Path:      "/private/auths/xai-demo.json",
		Status:    "active",
		Success:   7,
		Failed:    2,
		RecentRequests: []pluginapi.HostRecentRequestEntry{{
			Time: "10:00", Success: 3, Failed: 1,
		}},
	})
	raw, err := json.Marshal(summary)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "/private/auths") || strings.Contains(string(raw), "path") {
		t.Fatalf("summary leaked physical path: %s", raw)
	}
	if summary.AuthIndex != "auth-1" || summary.Success != 7 || len(summary.RecentRequests) != 1 {
		t.Fatalf("summary lost runtime fields: %+v", summary)
	}
}

func TestPathMatch(t *testing.T) {
	if !pathMatch("/v0/management/plugins/grok-sso2auth/convert", mgmtConvertPath) {
		t.Fatal("full path should match")
	}
	if !pathMatch(mgmtConvertPath, mgmtConvertPath) {
		t.Fatal("relative path should match")
	}
	if pathMatch("/v0/management/plugins/other/convert", mgmtConvertPath) {
		t.Fatal("other plugin should not match")
	}
	if !isResourceUIPath("/v0/resource/plugins/grok-sso2auth/status/") {
		t.Fatal("status resource should match")
	}
	if isResourceUIPath("/v0/resource/plugins/grok-sso2auth/") {
		t.Fatal("empty resource path should not match a registered route")
	}
}
