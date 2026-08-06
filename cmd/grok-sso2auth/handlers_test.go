package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
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
	if len(reg.Routes) != 2 {
		t.Fatalf("routes = %+v, want convert and convert-import only", reg.Routes)
	}
	for _, route := range reg.Routes {
		if route.Path != mgmtConvertPath && route.Path != mgmtConvertImportPath {
			t.Fatalf("unrelated management route remains: %+v", route)
		}
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
	for _, unwanted := range []string{`id="mgmtKey"`, "Auth 文件", "运行时详情", "导入已有 xAI OAuth JSON"} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("body still contains removed module %q", unwanted)
		}
	}
	for _, wanted := range []string{"已复用管理中心认证", "开始转换并导入", "cli-proxy-auth", "account_retries", "cli-proxy-theme", `data-theme="dark"`, `data-theme="white"`, "MutationObserver", "bootstrapBackground", "window.frameElement.style.backgroundColor", "parentReady", "resolvePreference"} {
		if !strings.Contains(body, wanted) {
			t.Fatalf("body missing %q", wanted)
		}
	}
	if !strings.Contains(body, `[hidden]{display:none!important}`) {
		t.Fatal("hidden result sections must stay hidden before the first run")
	}
	if !strings.Contains(body, `root.setAttribute("data-theme-source",inherited?"parent":"local")`) {
		t.Fatal("theme bridge must prefer the management center parent theme")
	}
	if strings.Contains(body, `transition:background-color`) {
		t.Fatal("background transitions cause a visible iframe theme flash")
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

func TestFlagDefaults(t *testing.T) {
	if got := numberFlag(nil, "missing", 45); got != 45 {
		t.Fatalf("number default = %v", got)
	}
	if got := numberFlag(map[string]any{"n": 7}, "n", 0); got != 7 {
		t.Fatalf("number int = %v", got)
	}
	if got := boolFlag(nil, "missing", true); !got {
		t.Fatal("validate SSO should default true")
	}
}
