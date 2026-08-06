package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type conversionJobItem struct {
	Index       int             `json:"index"`
	Email       string          `json:"email,omitempty"`
	FileName    string          `json:"file_name,omitempty"`
	Status      string          `json:"status"`
	OK          bool            `json:"ok"`
	Stage       string          `json:"stage"`
	Message     string          `json:"message"`
	Attempt     int             `json:"attempt,omitempty"`
	RateLimited bool            `json:"rate_limited,omitempty"`
	Error       string          `json:"error,omitempty"`
	Auth        json.RawMessage `json:"-"`
	SSO         string          `json:"-"`
}

type conversionJob struct {
	mu            sync.Mutex
	ID            string
	CreatedAt     time.Time
	State         string
	Items         []conversionJobItem
	OK            int
	Fail          int
	WorkerDone    bool
	BaseDelaySec  float64
	FinalDelaySec float64
	Request       convertRequest
}

type conversionJobResponse struct {
	JobID         string              `json:"job_id"`
	State         string              `json:"state"`
	Done          bool                `json:"done"`
	OK            int                 `json:"ok"`
	Fail          int                 `json:"fail"`
	Total         int                 `json:"total"`
	Items         []conversionJobItem `json:"items"`
	BaseDelaySec  float64             `json:"base_delay_sec"`
	FinalDelaySec float64             `json:"final_delay_sec"`
	ElapsedSec    int                 `json:"elapsed_sec"`
}

var conversionJobs = struct {
	sync.Mutex
	items map[string]*conversionJob
}{items: make(map[string]*conversionJob)}

func startConversionJob(body []byte) (int, any) {
	var req convertRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()}
	}
	entries := ParseSSOList(req.SSO)
	if len(entries) == 0 {
		return http.StatusBadRequest, map[string]string{"error": "sso is required"}
	}
	id, err := randomJobID()
	if err != nil {
		return http.StatusInternalServerError, map[string]string{"error": err.Error()}
	}
	pacer := newAdaptivePacer(req.BaseDelaySec, req.MaxDelaySec)
	job := &conversionJob{
		ID: id, CreatedAt: time.Now(), State: "running", Request: req,
		Items: make([]conversionJobItem, len(entries)), BaseDelaySec: pacer.Base(), FinalDelaySec: pacer.Current(),
	}
	for i, entry := range entries {
		job.Items[i] = conversionJobItem{Index: i + 1, Email: entry.Email, SSO: entry.SSO, Status: "queued", Stage: "queued", Message: "等待处理"}
	}
	conversionJobs.Lock()
	for key, existing := range conversionJobs.items {
		if time.Since(existing.CreatedAt) > 2*time.Hour {
			delete(conversionJobs.items, key)
		}
	}
	conversionJobs.items[id] = job
	conversionJobs.Unlock()
	go runConversionJob(job, pacer)
	return http.StatusAccepted, map[string]any{"job_id": id, "total": len(entries), "state": "running"}
}

func runConversionJob(job *conversionJob, pacer *adaptivePacer) {
	req := job.Request
	validate := true
	if req.ValidateSSO != nil {
		validate = *req.ValidateSSO
	}
	accountRetries := req.AccountRetries
	if accountRetries <= 0 {
		accountRetries = defaultAccountRetries
	}
	for i := range job.Items {
		var result *ConvertResult
		var convertErr error
		for attempt := 1; attempt <= accountRetries; attempt++ {
			job.update(i, func(item *conversionJobItem) {
				item.Status, item.Stage, item.Message, item.Attempt = "running", "validate", "开始处理账号", attempt
			})
			job.mu.Lock()
			sso, email := job.Items[i].SSO, firstNonEmpty(req.Email, job.Items[i].Email)
			job.mu.Unlock()
			result, convertErr = ConvertSSO(ConvertOptions{
				SSO: sso, Email: email, ValidateSSO: validate, MaxRetries: req.MaxRetries,
				BaseDelaySec: 15, PollTimeoutSec: req.PollTimeoutSec,
				Progress: func(progress ConversionProgress) {
					job.update(i, func(item *conversionJobItem) { item.Stage, item.Message = progress.Stage, progress.Message })
				},
			})
			if convertErr == nil || !isRateLimitedError(convertErr) {
				break
			}
			pacer.OnRateLimit()
			job.update(i, func(item *conversionJobItem) {
				item.RateLimited, item.Stage, item.Message = true, "retry_wait", fmt.Sprintf("遇到限流，准备第 %d 次账号级重试", attempt+1)
			})
			if attempt < accountRetries {
				time.Sleep(pacer.AccountRetryDelay(attempt))
			}
		}
		if convertErr != nil {
			job.mu.Lock()
			item := &job.Items[i]
			item.Status, item.Stage, item.Message, item.Error = "failed", "failed", "转换失败", convertErr.Error()
			job.Fail++
			job.FinalDelaySec = pacer.Current()
			job.mu.Unlock()
		} else {
			authJSON, _ := json.Marshal(result.Auth)
			job.mu.Lock()
			item := &job.Items[i]
			item.Email, item.FileName, item.Auth = result.Email, result.FileName, authJSON
			item.Status, item.Stage, item.Message = "pending_import", "import", "等待写入 CLIProxyAPI"
			job.mu.Unlock()
			pacer.OnSuccess()
		}
		if i < len(job.Items)-1 {
			job.update(i+1, func(item *conversionJobItem) { item.Stage, item.Message = "waiting", "等待账号间隔" })
			time.Sleep(pacer.BetweenAccountsDelay())
		}
	}
	job.mu.Lock()
	job.WorkerDone = true
	job.FinalDelaySec = pacer.Current()
	job.finishIfReadyLocked()
	job.mu.Unlock()
}

func pollConversionJob(id string) (int, any) {
	conversionJobs.Lock()
	job := conversionJobs.items[strings.TrimSpace(id)]
	conversionJobs.Unlock()
	if job == nil {
		return http.StatusNotFound, map[string]string{"error": "conversion job not found"}
	}
	type pendingImport struct {
		index int
		name  string
		auth  json.RawMessage
	}
	var pending []pendingImport
	job.mu.Lock()
	for i := range job.Items {
		if job.Items[i].Status == "pending_import" {
			job.Items[i].Status = "importing"
			pending = append(pending, pendingImport{i, job.Items[i].FileName, append(json.RawMessage(nil), job.Items[i].Auth...)})
		}
	}
	job.mu.Unlock()
	for _, item := range pending {
		saved, err := hostAuthSave(item.name, item.auth)
		job.mu.Lock()
		target := &job.Items[item.index]
		if err != nil {
			target.Status, target.Stage, target.Message, target.Error = "failed", "failed", "凭证写入失败", err.Error()
			job.Fail++
		} else {
			target.Status, target.Stage, target.Message, target.OK = "success", "done", "转换并导入成功", true
			target.FileName, target.Auth = saved.Name, nil
			job.OK++
		}
		job.finishIfReadyLocked()
		job.mu.Unlock()
	}
	return http.StatusOK, job.snapshot()
}

func (job *conversionJob) update(index int, fn func(*conversionJobItem)) {
	job.mu.Lock()
	fn(&job.Items[index])
	job.mu.Unlock()
}

func (job *conversionJob) finishIfReadyLocked() {
	if !job.WorkerDone {
		return
	}
	for _, item := range job.Items {
		if item.Status != "success" && item.Status != "failed" {
			return
		}
	}
	job.State = "completed"
}

func (job *conversionJob) snapshot() conversionJobResponse {
	job.mu.Lock()
	defer job.mu.Unlock()
	items := make([]conversionJobItem, len(job.Items))
	copy(items, job.Items)
	for i := range items {
		items[i].Auth, items[i].SSO = nil, ""
	}
	return conversionJobResponse{
		JobID: job.ID, State: job.State, Done: job.State == "completed", OK: job.OK, Fail: job.Fail,
		Total: len(items), Items: items, BaseDelaySec: job.BaseDelaySec, FinalDelaySec: job.FinalDelaySec,
		ElapsedSec: int(time.Since(job.CreatedAt).Seconds()),
	}
}

func randomJobID() (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate job id: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
