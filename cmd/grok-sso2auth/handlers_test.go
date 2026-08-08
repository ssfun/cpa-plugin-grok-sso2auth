package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

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
	if len(reg.Routes) != 7 {
		t.Fatalf("routes = %+v, want sync and observable job routes", reg.Routes)
	}
	for _, route := range reg.Routes {
		if route.Path != mgmtConvertPath && route.Path != mgmtConvertImportPath && route.Path != mgmtJobStartPath && route.Path != mgmtJobStatusPath && route.Path != mgmtJobPausePath && route.Path != mgmtJobResumePath && route.Path != mgmtJobTerminatePath {
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
	for _, wanted := range []string{"已复用管理中心认证", "开始转换并导入", "选择 TXT 文件", `accept=".txt,text/plain"`, "base_delay_min_sec", "base_delay_max_sec", `value="30"`, `value="3"`, "max-height:360px", "position:sticky", "cli-proxy-auth", "account_retries", "cli-proxy-theme", `data-theme="dark"`, `data-theme="white"`, "MutationObserver", "bootstrapBackground", "window.frameElement.style.backgroundColor", "parentReady", "resolvePreference", "restoreCurrentJob", "可安全切换或刷新页面", "暂停任务", "继续任务", "终止任务", "convert-job-pause", "convert-job-resume", "convert-job-terminate"} {
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

func TestConversionJobRequiresSSO(t *testing.T) {
	status, _ := startConversionJob([]byte(`{"sso":""}`))
	if status != http.StatusBadRequest {
		t.Fatalf("status=%d, want %d", status, http.StatusBadRequest)
	}
	conversionJobs.Lock()
	previous := conversionJobs.current
	conversionJobs.current = nil
	conversionJobs.Unlock()
	t.Cleanup(func() {
		conversionJobs.Lock()
		conversionJobs.current = previous
		conversionJobs.Unlock()
	})
	status, _ = currentConversionJobStatus()
	if status != http.StatusNotFound {
		t.Fatalf("missing job status=%d, want %d", status, http.StatusNotFound)
	}
}

func TestCurrentConversionJobStatusReturnsSingleJob(t *testing.T) {
	job := &conversionJob{
		ID: "current", CreatedAt: time.Now().Add(-3 * time.Second), State: "running",
		Items: []conversionJobItem{{Index: 1, Status: "running", Stage: "authorize", Message: "处理中", SSO: "secret"}},
	}
	conversionJobs.Lock()
	previous := conversionJobs.current
	conversionJobs.current = job
	conversionJobs.Unlock()
	t.Cleanup(func() {
		conversionJobs.Lock()
		conversionJobs.current = previous
		conversionJobs.Unlock()
	})

	status, payload := currentConversionJobStatus()
	if status != http.StatusOK {
		t.Fatalf("status=%d, want %d", status, http.StatusOK)
	}
	got, ok := payload.(conversionJobResponse)
	if !ok || got.JobID != "current" || got.Done || len(got.Items) != 1 {
		t.Fatalf("payload=%#v", payload)
	}
	if got.Items[0].SSO != "" {
		t.Fatal("status response leaked SSO")
	}
}

func TestStartConversionJobRejectsSecondRunningJob(t *testing.T) {
	job := newTestConversionJob("running")
	conversionJobs.Lock()
	previous := conversionJobs.current
	conversionJobs.current = job
	conversionJobs.Unlock()
	t.Cleanup(func() {
		conversionJobs.Lock()
		conversionJobs.current = previous
		conversionJobs.Unlock()
	})

	status, payload := startConversionJob([]byte(`{"sso":"test-sso"}`))
	if status != http.StatusConflict {
		t.Fatalf("status=%d, want %d; payload=%#v", status, http.StatusConflict, payload)
	}
	got, ok := payload.(map[string]any)
	if !ok || got["job"] == nil {
		t.Fatalf("conflict payload=%#v", payload)
	}
}

func TestPauseResumeAndTerminateConversionJob(t *testing.T) {
	job := newTestConversionJob("running")
	conversionJobs.Lock()
	previous := conversionJobs.current
	conversionJobs.current = job
	conversionJobs.Unlock()
	t.Cleanup(func() {
		conversionJobs.Lock()
		conversionJobs.current = previous
		conversionJobs.Unlock()
	})

	status, payload := controlCurrentConversionJob("pause")
	if status != http.StatusOK || payload.(conversionJobResponse).State != "paused" {
		t.Fatalf("pause status=%d payload=%#v", status, payload)
	}
	paused := make(chan error, 1)
	go func() { paused <- job.checkpoint() }()
	select {
	case err := <-paused:
		t.Fatalf("paused checkpoint returned early: %v", err)
	case <-time.After(20 * time.Millisecond):
	}

	status, payload = controlCurrentConversionJob("resume")
	if status != http.StatusOK || payload.(conversionJobResponse).State != "running" {
		t.Fatalf("resume status=%d payload=%#v", status, payload)
	}
	select {
	case err := <-paused:
		if err != nil {
			t.Fatalf("resumed checkpoint error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("resumed checkpoint stayed blocked")
	}

	status, payload = controlCurrentConversionJob("terminate")
	if status != http.StatusAccepted || payload.(conversionJobResponse).State != "terminating" {
		t.Fatalf("terminate status=%d payload=%#v", status, payload)
	}
	if !errors.Is(job.ctx.Err(), context.Canceled) {
		t.Fatalf("termination did not cancel context: %v", job.ctx.Err())
	}
}

func TestPausedJobSleepFreezesRemainingDelay(t *testing.T) {
	job := newTestConversionJob("running")
	done := make(chan error, 1)
	go func() { done <- job.sleep(90 * time.Millisecond) }()
	time.Sleep(20 * time.Millisecond)
	if status, _ := job.pause(); status != http.StatusOK {
		t.Fatalf("pause status=%d", status)
	}
	time.Sleep(100 * time.Millisecond)
	select {
	case err := <-done:
		t.Fatalf("sleep completed while paused: %v", err)
	default:
	}
	if status, _ := job.resumeJob(); status != http.StatusOK {
		t.Fatalf("resume status=%d", status)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("sleep error after resume: %v", err)
		}
	case <-time.After(150 * time.Millisecond):
		t.Fatal("sleep did not finish after resume")
	}
}

func TestShutdownCancelsAndWaitsForCurrentJob(t *testing.T) {
	job := newTestConversionJob("paused")
	conversionJobs.Lock()
	previousJob := conversionJobs.current
	previousShutdown := conversionJobs.shuttingDown
	conversionJobs.current = job
	conversionJobs.shuttingDown = false
	conversionJobs.Unlock()
	t.Cleanup(func() {
		conversionJobs.Lock()
		conversionJobs.current = previousJob
		conversionJobs.shuttingDown = previousShutdown
		conversionJobs.Unlock()
	})

	workerExited := make(chan struct{})
	go func() {
		<-job.ctx.Done()
		job.mu.Lock()
		job.State = "terminated"
		job.mu.Unlock()
		close(job.done)
		close(workerExited)
	}()

	shutdownConversionJobs()
	select {
	case <-workerExited:
	default:
		t.Fatal("shutdown returned before the worker exited")
	}
	if !errors.Is(job.ctx.Err(), context.Canceled) {
		t.Fatalf("shutdown did not cancel current job: %v", job.ctx.Err())
	}
	conversionJobs.Lock()
	shuttingDown := conversionJobs.shuttingDown
	conversionJobs.Unlock()
	if !shuttingDown {
		t.Fatal("shutdown did not reject future jobs")
	}
	if status, _ := startConversionJob([]byte(`{"sso":"test-sso"}`)); status != http.StatusServiceUnavailable {
		t.Fatalf("start during shutdown status=%d, want %d", status, http.StatusServiceUnavailable)
	}
}

func newTestConversionJob(state string) *conversionJob {
	ctx, cancel := context.WithCancel(context.Background())
	return &conversionJob{
		ID: "existing", CreatedAt: time.Now(), State: state,
		ctx: ctx, cancel: cancel, changed: make(chan struct{}), done: make(chan struct{}),
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

func TestConversionDefaults(t *testing.T) {
	if defaultBatchDelayMinSec != 3 || defaultBatchDelayMaxSec != 15 || defaultMaxDelaySec != 30 {
		t.Fatalf("delay defaults = %v-%v max %v", defaultBatchDelayMinSec, defaultBatchDelayMaxSec, defaultMaxDelaySec)
	}
	if defaultStageRetries != 3 || defaultAccountRetries != 3 {
		t.Fatalf("retry defaults = stage %d account %d", defaultStageRetries, defaultAccountRetries)
	}
}
