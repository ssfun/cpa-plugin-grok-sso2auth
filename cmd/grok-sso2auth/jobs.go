package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type conversionJobItem struct {
	Index       int    `json:"index"`
	Email       string `json:"email,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	Status      string `json:"status"`
	OK          bool   `json:"ok"`
	Stage       string `json:"stage"`
	Message     string `json:"message"`
	Attempt     int    `json:"attempt,omitempty"`
	RateLimited bool   `json:"rate_limited,omitempty"`
	Error       string `json:"error,omitempty"`
	SSO         string `json:"-"`
}

type conversionJob struct {
	mu              sync.Mutex
	ID              string
	CreatedAt       time.Time
	State           string
	Items           []conversionJobItem
	OK              int
	Fail            int
	BaseDelaySec    float64
	BaseDelayMinSec float64
	BaseDelayMaxSec float64
	FinalDelaySec   float64
	Request         convertRequest
	ctx             context.Context
	cancel          context.CancelFunc
	changed         chan struct{}
	done            chan struct{}
}

type conversionJobResponse struct {
	JobID           string              `json:"job_id"`
	State           string              `json:"state"`
	Done            bool                `json:"done"`
	OK              int                 `json:"ok"`
	Fail            int                 `json:"fail"`
	Total           int                 `json:"total"`
	Items           []conversionJobItem `json:"items"`
	BaseDelaySec    float64             `json:"base_delay_sec"`
	BaseDelayMinSec float64             `json:"base_delay_min_sec"`
	BaseDelayMaxSec float64             `json:"base_delay_max_sec"`
	FinalDelaySec   float64             `json:"final_delay_sec"`
	ElapsedSec      int                 `json:"elapsed_sec"`
}

var conversionJobs = struct {
	sync.Mutex
	current      *conversionJob
	shuttingDown bool
}{}

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
	pacer := newRequestPacer(req)
	ctx, cancel := context.WithCancel(context.Background())
	// The individual items own the SSO values while the worker is running.
	// Do not retain another copy in the job request.
	req.SSO = ""
	job := &conversionJob{
		ID: id, CreatedAt: time.Now(), State: "running", Request: req,
		Items: make([]conversionJobItem, len(entries)), BaseDelaySec: pacer.BaseMax(), BaseDelayMinSec: pacer.BaseMin(), BaseDelayMaxSec: pacer.BaseMax(), FinalDelaySec: pacer.Current(),
		ctx: ctx, cancel: cancel, changed: make(chan struct{}), done: make(chan struct{}),
	}
	for i, entry := range entries {
		job.Items[i] = conversionJobItem{Index: i + 1, Email: entry.Email, SSO: entry.SSO, Status: "queued", Stage: "queued", Message: "等待处理"}
	}
	conversionJobs.Lock()
	if conversionJobs.shuttingDown {
		conversionJobs.Unlock()
		cancel()
		return http.StatusServiceUnavailable, map[string]string{"error": "plugin is shutting down"}
	}
	if existing := conversionJobs.current; existing != nil {
		existing.mu.Lock()
		active := existing.State == "running" || existing.State == "paused" || existing.State == "terminating"
		existing.mu.Unlock()
		if active {
			conversionJobs.Unlock()
			cancel()
			return http.StatusConflict, map[string]any{
				"error": "a conversion job is already running",
				"job":   existing.snapshot(),
			}
		}
	}
	conversionJobs.current = job
	conversionJobs.Unlock()
	go runConversionJob(job, pacer)
	return http.StatusAccepted, map[string]any{"job_id": id, "total": len(entries), "state": "running"}
}

func shutdownConversionJobs() {
	conversionJobs.Lock()
	conversionJobs.shuttingDown = true
	job := conversionJobs.current
	conversionJobs.Unlock()
	if job == nil {
		return
	}

	job.mu.Lock()
	switch job.State {
	case "running", "paused":
		job.State = "terminating"
		job.cancel()
		close(job.changed)
		job.changed = make(chan struct{})
	}
	done := job.done
	job.mu.Unlock()
	if done != nil {
		<-done
	}
}

func (job *conversionJob) checkpoint() error {
	for {
		job.mu.Lock()
		state, changed := job.State, job.changed
		job.mu.Unlock()
		switch state {
		case "paused":
			select {
			case <-changed:
				continue
			case <-job.ctx.Done():
				return job.ctx.Err()
			}
		case "terminating", "terminated":
			return context.Canceled
		case "running":
			return job.ctx.Err()
		default:
			return fmt.Errorf("conversion job is not active")
		}
	}
}

func (job *conversionJob) sleep(delay time.Duration) error {
	remaining := delay
	for remaining > 0 {
		if err := job.checkpoint(); err != nil {
			return err
		}
		job.mu.Lock()
		state, changed := job.State, job.changed
		job.mu.Unlock()
		if state != "running" {
			continue
		}
		started := time.Now()
		timer := time.NewTimer(remaining)
		select {
		case <-timer.C:
			return job.checkpoint()
		case <-changed:
			if !timer.Stop() {
				<-timer.C
			}
			remaining -= time.Since(started)
		case <-job.ctx.Done():
			timer.Stop()
			return job.ctx.Err()
		}
	}
	return job.checkpoint()
}

func (job *conversionJob) pause() (int, any) {
	job.mu.Lock()
	defer job.mu.Unlock()
	switch job.State {
	case "running":
		job.State = "paused"
		close(job.changed)
		job.changed = make(chan struct{})
		return http.StatusOK, job.snapshotLocked()
	case "paused":
		return http.StatusOK, job.snapshotLocked()
	default:
		return http.StatusConflict, map[string]string{"error": "conversion job is not running"}
	}
}

func (job *conversionJob) resumeJob() (int, any) {
	job.mu.Lock()
	defer job.mu.Unlock()
	if job.State == "running" {
		return http.StatusOK, job.snapshotLocked()
	}
	if job.State != "paused" {
		return http.StatusConflict, map[string]string{"error": "conversion job is not paused"}
	}
	job.State = "running"
	close(job.changed)
	job.changed = make(chan struct{})
	return http.StatusOK, job.snapshotLocked()
}

func (job *conversionJob) terminate() (int, any) {
	job.mu.Lock()
	switch job.State {
	case "running", "paused":
		job.State = "terminating"
		job.cancel()
		close(job.changed)
		job.changed = make(chan struct{})
		response := job.snapshotLocked()
		job.mu.Unlock()
		return http.StatusAccepted, response
	case "terminating", "terminated":
		response := job.snapshotLocked()
		job.mu.Unlock()
		return http.StatusOK, response
	default:
		job.mu.Unlock()
		return http.StatusConflict, map[string]string{"error": "conversion job is already completed"}
	}
}

func controlCurrentConversionJob(action string) (int, any) {
	conversionJobs.Lock()
	job := conversionJobs.current
	conversionJobs.Unlock()
	if job == nil {
		return http.StatusNotFound, map[string]string{"error": "no conversion job"}
	}
	switch action {
	case "pause":
		return job.pause()
	case "resume":
		return job.resumeJob()
	case "terminate":
		return job.terminate()
	default:
		return http.StatusBadRequest, map[string]string{"error": "unknown conversion job action"}
	}
}

func runConversionJob(job *conversionJob, pacer *adaptivePacer) {
	defer job.finishWorker(pacer)
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
		if err := job.checkpoint(); err != nil {
			return
		}
		var result *ConvertResult
		var convertErr error
		job.mu.Lock()
		sso, email := job.Items[i].SSO, firstNonEmpty(req.Email, job.Items[i].Email)
		job.Items[i].SSO = ""
		job.mu.Unlock()
		for attempt := 1; attempt <= accountRetries; attempt++ {
			if err := job.checkpoint(); err != nil {
				return
			}
			job.update(i, func(item *conversionJobItem) {
				item.Status, item.Stage, item.Message, item.Attempt = "running", "validate", "开始处理账号", attempt
			})
			result, convertErr = ConvertSSO(ConvertOptions{
				Context: job.ctx,
				SSO:     sso, Email: email, ValidateSSO: validate, MaxRetries: req.MaxRetries,
				BaseDelaySec: 15, PollTimeoutSec: req.PollTimeoutSec,
				Checkpoint: job.checkpoint,
				Sleep:      job.sleep,
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
				if err := job.sleep(pacer.AccountRetryDelay(attempt)); err != nil {
					return
				}
			}
		}
		if errors.Is(convertErr, context.Canceled) {
			return
		}
		if convertErr != nil {
			job.mu.Lock()
			item := &job.Items[i]
			item.Status, item.Stage, item.Message, item.Error = "failed", "failed", "转换失败", convertErr.Error()
			job.Fail++
			job.FinalDelaySec = pacer.Current()
			job.mu.Unlock()
		} else {
			if err := job.checkpoint(); err != nil {
				return
			}
			authJSON, _ := json.Marshal(result.Auth)
			pacer.OnSuccess()
			job.mu.Lock()
			item := &job.Items[i]
			item.Email, item.FileName = result.Email, result.FileName
			item.Status, item.Stage, item.Message = "importing", "import", "正在写入 CLIProxyAPI"
			job.mu.Unlock()

			saved, saveErr := hostAuthSave(result.FileName, authJSON)
			job.mu.Lock()
			item = &job.Items[i]
			if saveErr != nil {
				item.Status, item.Stage, item.Message, item.Error = "failed", "failed", "凭证写入失败", saveErr.Error()
				job.Fail++
			} else {
				item.Status, item.Stage, item.Message, item.OK = "success", "done", "转换并导入成功", true
				item.FileName = saved.Name
				job.OK++
			}
			job.FinalDelaySec = pacer.Current()
			job.mu.Unlock()
		}
		if i < len(job.Items)-1 {
			job.update(i+1, func(item *conversionJobItem) { item.Stage, item.Message = "waiting", "等待账号间隔" })
			if err := job.sleep(pacer.BetweenAccountsDelay()); err != nil {
				return
			}
		}
	}
}

func (job *conversionJob) finishWorker(pacer *adaptivePacer) {
	if job.done != nil {
		defer close(job.done)
	}
	job.mu.Lock()
	job.FinalDelaySec = pacer.Current()
	terminated := job.State == "terminating" || job.ctx.Err() != nil
	if terminated {
		job.State = "terminated"
	} else {
		job.State = "completed"
	}
	job.Request = convertRequest{}
	for i := range job.Items {
		job.Items[i].SSO = ""
		if terminated && job.Items[i].Status != "success" && job.Items[i].Status != "failed" {
			job.Items[i].Status, job.Items[i].Stage, job.Items[i].Message = "terminated", "terminated", "任务已终止"
			job.Items[i].Error = "任务已终止"
			job.Fail++
		}
	}
	job.mu.Unlock()
}

func currentConversionJobStatus() (int, any) {
	conversionJobs.Lock()
	job := conversionJobs.current
	conversionJobs.Unlock()
	if job == nil {
		return http.StatusNotFound, map[string]string{"error": "no conversion job"}
	}
	return http.StatusOK, job.snapshot()
}

func (job *conversionJob) update(index int, fn func(*conversionJobItem)) {
	job.mu.Lock()
	fn(&job.Items[index])
	job.mu.Unlock()
}

func (job *conversionJob) snapshot() conversionJobResponse {
	job.mu.Lock()
	defer job.mu.Unlock()
	return job.snapshotLocked()
}

func (job *conversionJob) snapshotLocked() conversionJobResponse {
	items := make([]conversionJobItem, len(job.Items))
	copy(items, job.Items)
	for i := range items {
		items[i].SSO = ""
	}
	return conversionJobResponse{
		JobID: job.ID, State: job.State, Done: job.State == "completed" || job.State == "terminated", OK: job.OK, Fail: job.Fail,
		Total: len(items), Items: items, BaseDelaySec: job.BaseDelaySec, BaseDelayMinSec: job.BaseDelayMinSec, BaseDelayMaxSec: job.BaseDelayMaxSec, FinalDelaySec: job.FinalDelaySec,
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
