package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	mgmtImportPath        = "/plugins/grok-sso2auth/import"
	mgmtListPath          = "/plugins/grok-sso2auth/list"
	mgmtAuthRuntimePath   = "/plugins/grok-sso2auth/auth-runtime"
)

// Set via -ldflags "-X main.pluginVersion=..."
var pluginVersion = "0.2.0"

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
	Query          url.Values  `json:"Query"`
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
	ValidateSSO    bool    `json:"validate_sso,omitempty"`
	MaxRetries     int     `json:"max_retries,omitempty"`
	BaseDelaySec   float64 `json:"base_delay_sec,omitempty"`
	PollTimeoutSec int     `json:"poll_timeout_sec,omitempty"`
	// Import controls whether convert-import saves via host.auth.save.
	Import bool `json:"import,omitempty"`
	// DelaySec is the pause between multi-line accounts (convert batch).
	DelaySec float64 `json:"delay_sec,omitempty"`
}

type importRequest struct {
	// Name is the auth file name (must end with .json). Optional if auth.email/sub present.
	Name string          `json:"name,omitempty"`
	JSON json.RawMessage `json:"json"`
}

type batchItemResult struct {
	Index    int             `json:"index"`
	OK       bool            `json:"ok"`
	Email    string          `json:"email,omitempty"`
	FileName string          `json:"file_name,omitempty"`
	Path     string          `json:"path,omitempty"`
	Error    string          `json:"error,omitempty"`
	Auth     json.RawMessage `json:"auth,omitempty"`
}

type batchResponse struct {
	OK      int               `json:"ok"`
	Fail    int               `json:"fail"`
	Total   int               `json:"total"`
	Items   []batchItemResult `json:"items"`
	Message string            `json:"message,omitempty"`
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
				{Method: http.MethodPost, Path: mgmtImportPath, Description: "直接导入已有 xai auth JSON"},
				{Method: http.MethodGet, Path: mgmtListPath, Description: "列出当前 auth 文件"},
				{Method: http.MethodGet, Path: mgmtAuthRuntimePath, Description: "读取指定 auth 的运行时摘要"},
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
					"Usage":        "Seconds between accounts when using --grok-sso-file (default 45)",
					"Type":         "float64",
					"DefaultValue": 45,
				},
				{
					"Name":         "grok-sso-validate",
					"Usage":        "Validate SSO against accounts.x.ai before device flow",
					"Type":         "bool",
					"DefaultValue": false,
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
			ConfigFields: []pluginapi.ConfigField{
				{
					Name:        "default_delay_sec",
					Type:        pluginapi.ConfigFieldTypeNumber,
					Description: "Default seconds between batch accounts (default 45).",
				},
				{
					Name:        "validate_sso",
					Type:        pluginapi.ConfigFieldTypeBoolean,
					Description: "Pre-validate SSO cookie via accounts.x.ai (extra request).",
				},
			},
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
	case method == http.MethodPost && pathMatch(p, mgmtImportPath):
		return okEnvelope(jsonAPIResponse(handleImport(req.Body)))
	case method == http.MethodGet && pathMatch(p, mgmtListPath):
		return okEnvelope(jsonAPIResponse(handleList()))
	case method == http.MethodGet && pathMatch(p, mgmtAuthRuntimePath):
		return okEnvelope(jsonAPIResponse(handleAuthRuntime(req.Query)))
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

	delay := req.DelaySec
	if delay < 0 {
		delay = 0
	}
	if delay == 0 && len(entries) > 1 {
		delay = 45
	}

	resp := batchResponse{Total: len(entries), Items: make([]batchItemResult, 0, len(entries))}
	for i, ent := range entries {
		item := batchItemResult{Index: i + 1, Email: ent.Email}
		email := firstNonEmpty(req.Email, ent.Email)
		result, err := ConvertSSO(ConvertOptions{
			SSO:            ent.SSO,
			Email:          email,
			ValidateSSO:    req.ValidateSSO,
			MaxRetries:     req.MaxRetries,
			BaseDelaySec:   req.BaseDelaySec,
			PollTimeoutSec: req.PollTimeoutSec,
		})
		if err != nil {
			item.Error = err.Error()
			resp.Fail++
			resp.Items = append(resp.Items, item)
			if i < len(entries)-1 && delay > 0 {
				time.Sleep(time.Duration(delay * float64(time.Second)))
			}
			continue
		}
		item.OK = true
		item.Email = result.Email
		item.FileName = result.FileName
		authJSON, _ := json.Marshal(result.Auth)
		item.Auth = authJSON

		if doImport {
			saved, errSave := hostAuthSave(result.FileName, authJSON)
			if errSave != nil {
				item.OK = false
				item.Error = "converted but import failed: " + errSave.Error()
				resp.Fail++
			} else {
				item.Path = saved.Path
				item.FileName = saved.Name
				resp.OK++
			}
		} else {
			resp.OK++
		}
		resp.Items = append(resp.Items, item)
		if i < len(entries)-1 && delay > 0 {
			time.Sleep(time.Duration(delay * float64(time.Second)))
		}
	}
	resp.Message = fmt.Sprintf("%d/%d succeeded", resp.OK, resp.Total)
	status := http.StatusOK
	if resp.OK == 0 {
		status = http.StatusBadGateway
	}
	return status, resp
}

func handleImport(body []byte) (int, any) {
	var req importRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()}
	}
	if len(req.JSON) == 0 || string(req.JSON) == "null" {
		return http.StatusBadRequest, map[string]string{"error": "json is required"}
	}
	var probe struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(req.JSON, &probe); err != nil {
		return http.StatusBadRequest, map[string]string{"error": "invalid auth json: " + err.Error()}
	}
	if !strings.EqualFold(strings.TrimSpace(probe.Type), "xai") {
		return http.StatusBadRequest, map[string]string{"error": "auth json type must be xai"}
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		// derive from payload
		name = credentialFileName(probe.Email, probe.Sub)
	}
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		name += ".json"
	}
	saved, err := hostAuthSave(name, req.JSON)
	if err != nil {
		return http.StatusBadGateway, map[string]string{"error": err.Error()}
	}
	return http.StatusOK, map[string]any{
		"ok":   true,
		"name": saved.Name,
		"path": saved.Path,
	}
}

func handleList() (int, any) {
	result, err := callHost(pluginabi.MethodHostAuthList, map[string]any{})
	if err != nil {
		return http.StatusBadGateway, map[string]string{"error": err.Error()}
	}
	var list struct {
		Files []pluginapi.HostAuthFileEntry `json:"files"`
	}
	if err := json.Unmarshal(result, &list); err != nil {
		return http.StatusBadGateway, map[string]string{"error": "decode list: " + err.Error()}
	}
	// Strip sensitive runtime fields such as physical paths. The host callback
	// intentionally exposes a richer entry than a browser list needs.
	summaries := make([]authSummary, 0, len(list.Files))
	for _, f := range list.Files {
		summaries = append(summaries, summarizeAuth(f))
	}
	return http.StatusOK, map[string]any{"files": summaries, "count": len(summaries)}
}

// authSummary is the safe browser-facing subset of pluginapi.HostAuthFileEntry.
// In particular, it omits Path and any credential JSON returned by host.auth.get.
type authSummary struct {
	ID             string                 `json:"id,omitempty"`
	AuthIndex      string                 `json:"auth_index,omitempty"`
	Name           string                 `json:"name"`
	Type           string                 `json:"type,omitempty"`
	Provider       string                 `json:"provider,omitempty"`
	Label          string                 `json:"label,omitempty"`
	Status         string                 `json:"status,omitempty"`
	StatusMessage  string                 `json:"status_message,omitempty"`
	Disabled       bool                   `json:"disabled,omitempty"`
	Unavailable    bool                   `json:"unavailable,omitempty"`
	RuntimeOnly    bool                   `json:"runtime_only,omitempty"`
	Source         string                 `json:"source,omitempty"`
	Size           int64                  `json:"size,omitempty"`
	ModTime        time.Time              `json:"modtime,omitempty"`
	UpdatedAt      time.Time              `json:"updated_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at,omitempty"`
	LastRefresh    time.Time              `json:"last_refresh,omitempty"`
	NextRetryAfter time.Time              `json:"next_retry_after,omitempty"`
	Email          string                 `json:"email,omitempty"`
	ProjectID      string                 `json:"project_id,omitempty"`
	AccountType    string                 `json:"account_type,omitempty"`
	Account        string                 `json:"account,omitempty"`
	Priority       int                    `json:"priority,omitempty"`
	Note           string                 `json:"note,omitempty"`
	Websockets     bool                   `json:"websockets,omitempty"`
	Success        int64                  `json:"success,omitempty"`
	Failed         int64                  `json:"failed,omitempty"`
	RecentRequests []recentRequestSummary `json:"recent_requests,omitempty"`
}

type recentRequestSummary struct {
	Time    string `json:"time"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
}

func summarizeAuth(f pluginapi.HostAuthFileEntry) authSummary {
	result := authSummary{
		ID:             f.ID,
		AuthIndex:      f.AuthIndex,
		Name:           f.Name,
		Type:           f.Type,
		Provider:       f.Provider,
		Label:          f.Label,
		Status:         f.Status,
		StatusMessage:  f.StatusMessage,
		Disabled:       f.Disabled,
		Unavailable:    f.Unavailable,
		RuntimeOnly:    f.RuntimeOnly,
		Source:         f.Source,
		Size:           f.Size,
		ModTime:        f.ModTime,
		UpdatedAt:      f.UpdatedAt,
		CreatedAt:      f.CreatedAt,
		LastRefresh:    f.LastRefresh,
		NextRetryAfter: f.NextRetryAfter,
		Email:          f.Email,
		ProjectID:      f.ProjectID,
		AccountType:    f.AccountType,
		Account:        f.Account,
		Priority:       f.Priority,
		Note:           f.Note,
		Websockets:     f.Websockets,
		Success:        f.Success,
		Failed:         f.Failed,
	}
	if len(f.RecentRequests) > 0 {
		result.RecentRequests = make([]recentRequestSummary, 0, len(f.RecentRequests))
		for _, item := range f.RecentRequests {
			result.RecentRequests = append(result.RecentRequests, recentRequestSummary{
				Time: item.Time, Success: item.Success, Failed: item.Failed,
			})
		}
	}
	return result
}

func handleAuthRuntime(query url.Values) (int, any) {
	authIndex := strings.TrimSpace(query.Get("auth_index"))
	if authIndex == "" {
		return http.StatusBadRequest, map[string]string{"error": "auth_index is required"}
	}
	result, err := callHost(pluginabi.MethodHostAuthGetRuntime, pluginapi.HostAuthGetRequest{
		AuthIndex: authIndex,
	})
	if err != nil {
		status := http.StatusBadGateway
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			status = http.StatusNotFound
		}
		return status, map[string]string{"error": err.Error()}
	}
	var runtimeInfo pluginapi.HostAuthGetRuntimeResponse
	if err := json.Unmarshal(result, &runtimeInfo); err != nil {
		return http.StatusBadGateway, map[string]string{"error": "decode runtime info: " + err.Error()}
	}
	return http.StatusOK, map[string]any{
		"auth": summarizeAuth(runtimeInfo.Auth),
	}
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
	validate, _ := flags["grok-sso-validate"].(bool)
	delay := 45.0
	if v, ok := flags["grok-sso-delay"]; ok {
		switch t := v.(type) {
		case float64:
			delay = t
		case int:
			delay = float64(t)
		}
	}

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
		SSO:         rawSSO,
		Email:       email,
		ValidateSSO: validate,
		DelaySec:    delay,
		Import:      true,
	})
	status, payload := handleConvert(body, true)
	pretty, _ := json.MarshalIndent(payload, "", "  ")
	exit := 0
	if status >= 400 {
		exit = 1
	}
	return okEnvelope(cliResult(exit, string(pretty)+"\n", ""))
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
