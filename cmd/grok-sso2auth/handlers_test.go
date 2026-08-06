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
	if len(reg.Routes) < 3 {
		t.Fatalf("routes = %+v", reg.Routes)
	}
}

func TestManagementUIServesHTML(t *testing.T) {
	req, _ := json.Marshal(managementRequest{
		Method: http.MethodGet,
		Path:   "/v0/resource/plugins/grok-sso2auth/",
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
}
