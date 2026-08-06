package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginabi"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	pluginID   = "grok-sso2auth"
	pluginName = "Grok SSO → Auth"
	// pluginVersion is overridden at link time:
	//   -ldflags "-X main.pluginVersion=1.2.3"
	pluginAuthor = "sfun"
	pluginRepo   = "https://github.com/ssfun/cpa-plugin-grok-sso2auth"

	// Resource paths are normalized by the CLIProxyAPI host and must not be
	// empty after trimming the trailing slash. Keep the page on a named route
	// instead of registering "/", which the host rejects.
	resourceUIPath = "/status"
	// Management API paths (relative; host mounts under /v0/management).
	mgmtConvertPath       = "/plugins/grok-sso2auth/convert"
	mgmtConvertImportPath = "/plugins/grok-sso2auth/convert-import"

	defaultBatchDelaySec  = 45.0
	defaultMaxDelaySec    = 180.0
	defaultAccountRetries = 3
)

// Set via -ldflags "-X main.pluginVersion=..."
var pluginVersion = "0.3.1"

type registration struct {
	SchemaVersion uint32             `json:"schema_version"`
	Metadata      pluginapi.Metadata `json:"metadata"`
	Capabilities  regCapabilities    `json:"capabilities"`
}

type regCapabilities struct {
	ManagementAPI     bool `json:"management_api"`
	CommandLinePlugin bool `json:"command_line_plugin"`
}

type managementRegistration struct {
	Routes    []managementRoute    `json:"routes,omitempty"`
	Resources []managementResource `json:"resources,omitempty"`
}

type managementRoute struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Menu        string `json:"menu,omitempty"`
	Description string `json:"description,omitempty"`
}

type managementResource struct {
	Path        string `json:"path"`
	Menu        string `json:"menu"`
	Description string `json:"description"`
}

type managementRequest struct {
	Method         string      `json:"Method"`
	Path           string      `json:"Path"`
	Headers        http.Header `json:"Headers"`
	Body           []byte      `json:"Body"`
	HostCallbackID string      `json:"host_callback_id,omitempty"`
}

type managementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}

type convertRequest struct {
	SSO            string  `json:"sso"`
	Email          string  `json:"email,omitempty"`
	ValidateSSO    *bool   `json:"validate_sso,omitempty"`
	MaxRetries     int     `json:"max_retries,omitempty"`
	AccountRetries int     `json:"account_retries,omitempty"`
	BaseDelaySec   float64 `json:"base_delay_sec,omitempty"`
	MaxDelaySec    float64 `json:"max_delay_sec,omitempty"`
	PollTimeoutSec int     `json:"poll_timeout_sec,omitempty"`
}

type batchItemResult struct {
	Index       int             `json:"index"`
	OK          bool            `json:"ok"`
	Email       string          `json:"email,omitempty"`
	FileName    string          `json:"file_name,omitempty"`
	Error       string          `json:"error,omitempty"`
	Auth        json.RawMessage `json:"auth,omitempty"`
	Attempts    int             `json:"attempts,omitempty"`
	RateLimited bool            `json:"rate_limited,omitempty"`
}

type batchResponse struct {
	OK            int               `json:"ok"`
	Fail          int               `json:"fail"`
	Total         int               `json:"total"`
	Items         []batchItemResult `json:"items"`
	Message       string            `json:"message,omitempty"`
	Imported      bool              `json:"imported"`
	BaseDelaySec  float64           `json:"base_delay_sec,omitempty"`
	FinalDelaySec float64           `json:"final_delay_sec,omitempty"`
}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case pluginabi.MethodPluginRegister, pluginabi.MethodPluginReconfigure:
		return okEnvelope(pluginRegistration())
	case pluginabi.MethodManagementRegister:
		return okEnvelope(managementRegistration{
			Resources: []managementResource{{
				Path:        resourceUIPath,
				Menu:        "Grok SSO 导入",
				Description: "将 Grok/xAI SSO Cookie 转换为 CLIProxyAPI xai OAuth 凭证并导入 auth-dir。",
			}},
			Routes: []managementRoute{
				{Method: http.MethodPost, Path: mgmtConvertPath, Description: "SSO → xai auth JSON（不写入）"},
				{Method: http.MethodPost, Path: mgmtConvertImportPath, Description: "SSO → xai auth JSON 并 host.auth.save 导入"},
			},
		})
	case pluginabi.MethodManagementHandle:
		return handleManagement(request)
	case pluginabi.MethodCommandLineRegister:
		return okEnvelope(map[string]any{
			"Flags": []map[string]any{
				{
					"Name":  "grok-sso-cookie",
					"Usage": "Convert one Grok SSO cookie to xai auth and import via host.auth.save",
					"Type":  "string",
				},
				{
					"Name":  "grok-sso-file",
					"Usage": "Convert SSO list file (one JWT or email----sso per line) and import",
					"Type":  "string",
				},
				{
					"Name":         "grok-sso-email",
					"Usage":        "Optional email override for --grok-sso-cookie",
					"Type":         "string",
					"DefaultValue": "",
				},
				{
					"Name":         "grok-sso-delay",
					"Usage":        "Base seconds between accounts; adaptive pacing starts at 45",
					"Type":         "float64",
					"DefaultValue": defaultBatchDelaySec,
				},
				{
					"Name":         "grok-sso-max-delay",
					"Usage":        "Maximum adaptive delay between accounts (default 180)",
					"Type":         "float64",
					"DefaultValue": defaultMaxDelaySec,
				},
				{
					"Name":         "grok-sso-retries",
					"Usage":        "Maximum device/verify/approve retries per account (default 8)",
					"Type":         "int",
					"DefaultValue": 8,
				},
				{
					"Name":         "grok-sso-account-retries",
					"Usage":        "Whole-account retries after rate limiting (default 3)",
					"Type":         "int",
					"DefaultValue": defaultAccountRetries,
				},
				{
					"Name":         "grok-sso-validate",
					"Usage":        "Validate SSO against accounts.x.ai before device flow",
					"Type":         "bool",
					"DefaultValue": true,
				},
			},
		})
	case pluginabi.MethodCommandLineExecute:
		return handleCommandLine(request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: pluginabi.SchemaVersion,
		Metadata: pluginapi.Metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           pluginAuthor,
			GitHubRepository: pluginRepo,
		},
		Capabilities: regCapabilities{
			ManagementAPI:     true,
			CommandLinePlugin: true,
		},
	}
}

func handleManagement(raw []byte) ([]byte, error) {
	var req managementRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, fmt.Errorf("decode management request: %w", err)
		}
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	p := strings.TrimSpace(req.Path)

	// Resource UI (unauthenticated page delivery).
	if method == http.MethodGet && isResourceUIPath(p) {
		return okEnvelope(htmlResponse(http.StatusOK, uiHTML()))
	}

	// Management API routes (management-key protected by host).
	switch {
	case method == http.MethodPost && pathMatch(p, mgmtConvertPath):
		return okEnvelope(jsonAPIResponse(handleConvert(req.Body, false)))
	case method == http.MethodPost && pathMatch(p, mgmtConvertImportPath):
		return okEnvelope(jsonAPIResponse(handleConvert(req.Body, true)))
	default:
		body, _ := json.Marshal(map[string]any{
			"error":  "not_found",
			"method": method,
			"path":   p,
		})
		return okEnvelope(managementResponse{
			StatusCode: http.StatusNotFound,
			Headers:    http.Header{"Content-Type": []string{"application/json"}},
			Body:       body,
		})
	}
}

func isResourceUIPath(p string) bool {
	p = strings.TrimRight(p, "/")
	if p == "" {
		return false
	}
	// Full: /v0/resource/plugins/grok-sso2auth/status or the legacy
	// relative form /plugins/grok-sso2auth/status.
	return strings.HasSuffix(p, "/v0/resource/plugins/"+pluginID+resourceUIPath) ||
		strings.HasSuffix(p, "/plugins/"+pluginID+resourceUIPath) ||
		p == resourceUIPath
}

func pathMatch(full, rel string) bool {
	full = strings.TrimRight(full, "/")
	rel = strings.TrimRight(rel, "/")
	if full == rel {
		return true
	}
	if strings.HasSuffix(full, rel) {
		return true
	}
	// Also accept without leading /plugins prefix variations
	return strings.HasSuffix(full, strings.TrimPrefix(rel, "/"))
}

func htmlResponse(status int, body []byte) managementResponse {
	return managementResponse{
		StatusCode: status,
		Headers: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
		Body: body,
	}
}

func jsonAPIResponse(status int, payload any) managementResponse {
	body, err := json.Marshal(payload)
	if err != nil {
		body, _ = json.Marshal(map[string]string{"error": err.Error()})
		status = http.StatusInternalServerError
	}
	return managementResponse{
		StatusCode: status,
		Headers: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
		},
		Body: body,
	}
}

func handleConvert(body []byte, doImport bool) (int, any) {
	var req convertRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()}
		}
	}
	entries := ParseSSOList(req.SSO)
	if len(entries) == 0 && strings.TrimSpace(req.SSO) != "" {
		// single raw cookie without newline
		entries = ParseSSOList(req.SSO + "\n")
	}
	if len(entries) == 0 {
		return http.StatusBadRequest, map[string]string{"error": "sso is required (cookie or multi-line list)"}
	}

	validateSSO := true
	if req.ValidateSSO != nil {
		validateSSO = *req.ValidateSSO
	}
	accountRetries := req.AccountRetries
	if accountRetries <= 0 {
		accountRetries = defaultAccountRetries
	}
	pacer := newAdaptivePacer(req.BaseDelaySec, req.MaxDelaySec)
	resp := batchResponse{
		Total:        len(entries),
		Items:        make([]batchItemResult, 0, len(entries)),
		Imported:     doImport,
		BaseDelaySec: pacer.Base(),
	}
	for i, ent := range entries {
		item := batchItemResult{Index: i + 1, Email: ent.Email}
		email := firstNonEmpty(req.Email, ent.Email)
		var result *ConvertResult
		var err error
		for attempt := 1; attempt <= accountRetries; attempt++ {
			item.Attempts = attempt
			result, err = ConvertSSO(ConvertOptions{
				SSO:            ent.SSO,
				Email:          email,
				ValidateSSO:    validateSSO,
				MaxRetries:     req.MaxRetries,
				BaseDelaySec:   15,
				PollTimeoutSec: req.PollTimeoutSec,
			})
			if err == nil || !isRateLimitedError(err) {
				break
			}
			item.RateLimited = true
			pacer.OnRateLimit()
			if attempt < accountRetries {
				time.Sleep(pacer.AccountRetryDelay(attempt))
			}
		}
		if err != nil {
			item.Error = err.Error()
			resp.Fail++
			resp.Items = append(resp.Items, item)
			if item.RateLimited {
				pacer.OnRateLimit()
			}
			if i < len(entries)-1 {
				time.Sleep(pacer.BetweenAccountsDelay())
			}
			continue
		}
		item.OK = true
		item.Email = result.Email
		item.FileName = result.FileName
		authJSON, _ := json.Marshal(result.Auth)
		if !doImport {
			item.Auth = authJSON
		}

		if doImport {
			saved, errSave := hostAuthSave(result.FileName, authJSON)
			if errSave != nil {
				item.OK = false
				item.Error = "converted but import failed: " + errSave.Error()
				resp.Fail++
			} else {
				item.FileName = saved.Name
				resp.OK++
			}
		} else {
			resp.OK++
		}
		resp.Items = append(resp.Items, item)
		pacer.OnSuccess()
		if i < len(entries)-1 {
			time.Sleep(pacer.BetweenAccountsDelay())
		}
	}
	resp.FinalDelaySec = pacer.Current()
	resp.Message = fmt.Sprintf("%d/%d succeeded", resp.OK, resp.Total)
	status := http.StatusOK
	if resp.OK == 0 {
		status = http.StatusBadGateway
	}
	return status, resp
}

func hostAuthSave(name string, rawJSON json.RawMessage) (pluginapi.HostAuthSaveResponse, error) {
	result, err := callHost(pluginabi.MethodHostAuthSave, pluginapi.HostAuthSaveRequest{
		Name: name,
		JSON: rawJSON,
	})
	if err != nil {
		return pluginapi.HostAuthSaveResponse{}, err
	}
	var resp pluginapi.HostAuthSaveResponse
	if err := json.Unmarshal(result, &resp); err != nil {
		return pluginapi.HostAuthSaveResponse{}, fmt.Errorf("decode host.auth.save: %w", err)
	}
	return resp, nil
}

// ---- command line ----

type commandLineExecutionRequest struct {
	Program        string            `json:"Program"`
	Args           []string          `json:"Args"`
	ConfigPath     string            `json:"ConfigPath"`
	Flags          []commandLineFlag `json:"Flags"`
	TriggeredFlags []commandLineFlag `json:"TriggeredFlags"`
}

type commandLineFlag struct {
	Name  string `json:"Name"`
	Type  string `json:"Type"`
	Value any    `json:"Value"`
	Set   bool   `json:"Set"`
}

func handleCommandLine(raw []byte) ([]byte, error) {
	var req commandLineExecutionRequest
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &req)
	}
	flags := indexFlags(req.TriggeredFlags)
	if len(flags) == 0 {
		flags = indexFlags(req.Flags)
	}

	cookie, _ := flags["grok-sso-cookie"].(string)
	filePath, _ := flags["grok-sso-file"].(string)
	email, _ := flags["grok-sso-email"].(string)
	validate := boolFlag(flags, "grok-sso-validate", true)
	delay := numberFlag(flags, "grok-sso-delay", defaultBatchDelaySec)
	maxDelay := numberFlag(flags, "grok-sso-max-delay", defaultMaxDelaySec)
	maxRetries := int(numberFlag(flags, "grok-sso-retries", 8))
	accountRetries := int(numberFlag(flags, "grok-sso-account-retries", defaultAccountRetries))

	var rawSSO string
	if strings.TrimSpace(filePath) != "" {
		// Read via os — plugin runs in-process with host FS access.
		data, err := readFileLimited(filePath, 8<<20)
		if err != nil {
			return okEnvelope(cliResult(1, "", "read sso file: "+err.Error()))
		}
		rawSSO = string(data)
	} else if strings.TrimSpace(cookie) != "" {
		rawSSO = cookie
	} else {
		return okEnvelope(cliResult(0, "grok-sso2auth: no --grok-sso-cookie / --grok-sso-file set\n", ""))
	}

	body, _ := json.Marshal(convertRequest{
		SSO:            rawSSO,
		Email:          email,
		ValidateSSO:    &validate,
		BaseDelaySec:   delay,
		MaxDelaySec:    maxDelay,
		MaxRetries:     maxRetries,
		AccountRetries: accountRetries,
	})
	status, payload := handleConvert(body, true)
	pretty, _ := json.MarshalIndent(payload, "", "  ")
	exit := 0
	if status >= 400 {
		exit = 1
	}
	return okEnvelope(cliResult(exit, string(pretty)+"\n", ""))
}

func numberFlag(flags map[string]any, name string, fallback float64) float64 {
	v, ok := flags[name]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		if parsed, err := n.Float64(); err == nil {
			return parsed
		}
	}
	return fallback
}

func boolFlag(flags map[string]any, name string, fallback bool) bool {
	v, ok := flags[name]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}

func indexFlags(flags []commandLineFlag) map[string]any {
	out := map[string]any{}
	for _, f := range flags {
		if !f.Set && f.Value == nil {
			continue
		}
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		out[name] = f.Value
	}
	return out
}

func cliResult(exit int, stdout, stderr string) map[string]any {
	return map[string]any{
		"Stdout":   []byte(stdout),
		"Stderr":   []byte(stderr),
		"ExitCode": exit,
	}
}

func readFileLimited(path string, limit int64) ([]byte, error) {
	// local import to keep file IO in one place
	return readFileLimitedImpl(path, limit)
}
