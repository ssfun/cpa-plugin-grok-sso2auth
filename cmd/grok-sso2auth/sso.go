package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

const (
	xaiClientID         = "b1a00492-073a-47ea-816f-4c329264a828"
	xaiIssuer           = "https://auth.x.ai"
	xaiScope            = "openid profile email offline_access grok-cli:access api:access conversations:read conversations:write"
	xaiDefaultAPIBase   = "https://api.x.ai/v1"
	xaiTokenEndpoint    = xaiIssuer + "/oauth2/token"
	xaiDeviceCodeURL    = xaiIssuer + "/oauth2/device/code"
	xaiDeviceVerifyURL  = xaiIssuer + "/oauth2/device/verify"
	xaiDeviceApproveURL = xaiIssuer + "/oauth2/device/approve"
	xaiUserinfoURL      = xaiIssuer + "/oauth2/userinfo"
	xaiRedirectURI      = "http://127.0.0.1:56121/callback"
	xaiAccountsURL      = "https://accounts.x.ai/"
	httpTimeout         = 20 * time.Second
	defaultPollInterval = 5 * time.Second
	pollTimeout         = 90 * time.Second
)

// TokenPayload is the raw OAuth token response + optional email enrichment.
type TokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Email        string `json:"email,omitempty"`
	Subject      string `json:"sub,omitempty"`
}

// XAIAuthFile is the on-disk credential format expected by CLIProxyAPI (type=xai).
// Mirrors internal/auth/xai.TokenStorage.
type XAIAuthFile struct {
	Type          string `json:"type"`
	AuthKind      string `json:"auth_kind,omitempty"`
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	IDToken       string `json:"id_token,omitempty"`
	TokenType     string `json:"token_type,omitempty"`
	ExpiresIn     int    `json:"expires_in,omitempty"`
	Expired       string `json:"expired,omitempty"`
	LastRefresh   string `json:"last_refresh,omitempty"`
	Email         string `json:"email,omitempty"`
	Subject       string `json:"sub,omitempty"`
	BaseURL       string `json:"base_url,omitempty"`
	TokenEndpoint string `json:"token_endpoint,omitempty"`
	RedirectURI   string `json:"redirect_uri,omitempty"`
	Disabled      bool   `json:"disabled"`
}

// ConvertOptions controls SSO → token conversion.
type ConvertOptions struct {
	SSO            string
	Email          string
	ValidateSSO    bool
	MaxRetries     int
	BaseDelaySec   float64
	PollTimeoutSec int
}

// ConvertResult is one successful conversion.
type ConvertResult struct {
	FileName string      `json:"file_name"`
	Email    string      `json:"email,omitempty"`
	Subject  string      `json:"sub,omitempty"`
	Auth     XAIAuthFile `json:"auth"`
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type rateLimitedError struct {
	msg string
}

func (e *rateLimitedError) Error() string { return e.msg }

func isRateLimitedError(err error) bool {
	var target *rateLimitedError
	return errors.As(err, &target)
}

type adaptivePacer struct {
	base    float64
	current float64
	max     float64
	hits    int
}

func newAdaptivePacer(base, maxDelay float64) *adaptivePacer {
	if base <= 0 {
		base = defaultBatchDelaySec
	}
	if maxDelay < 30 {
		maxDelay = defaultMaxDelaySec
	}
	if maxDelay < base {
		maxDelay = base
	}
	return &adaptivePacer{base: base, current: base, max: maxDelay}
}

func (p *adaptivePacer) Base() float64    { return p.base }
func (p *adaptivePacer) Current() float64 { return p.current }

func (p *adaptivePacer) OnRateLimit() {
	p.hits++
	next := maxFloat(p.current*1.8, p.current+25, defaultBatchDelaySec)
	p.current = minFloat(next, p.max)
}

func (p *adaptivePacer) OnSuccess() {
	if p.hits > 0 {
		p.hits--
	}
	if p.current > p.base {
		p.current = maxFloat(p.base, p.current*0.92)
	}
}

func (p *adaptivePacer) AccountRetryDelay(attempt int) time.Duration {
	return backoff(p.current, attempt, p.max)
}

func (p *adaptivePacer) BetweenAccountsDelay() time.Duration {
	jitterNanos := time.Now().UnixNano() % 11
	if jitterNanos < 0 {
		jitterNanos = -jitterNanos
	}
	jitter := float64(jitterNanos)
	return time.Duration((p.current + jitter) * float64(time.Second))
}

func newHTTPClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: httpTimeout,
		Jar:     jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			// Keep cookies across redirects; default client already does via Jar.
			return nil
		},
	}
}

func setSSOCookie(client *http.Client, sso string) {
	sso = strings.TrimSpace(sso)
	u, _ := url.Parse("https://x.ai/")
	client.Jar.SetCookies(u, []*http.Cookie{{
		Name:   "sso",
		Value:  sso,
		Domain: ".x.ai",
		Path:   "/",
	}})
	// Also set for auth.x.ai / accounts.x.ai hosts explicitly.
	for _, host := range []string{"https://auth.x.ai/", "https://accounts.x.ai/"} {
		hu, _ := url.Parse(host)
		client.Jar.SetCookies(hu, []*http.Cookie{{
			Name:   "sso",
			Value:  sso,
			Domain: ".x.ai",
			Path:   "/",
		}})
	}
}

func chromeHeaders() http.Header {
	h := make(http.Header)
	h.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	h.Set("Accept", "application/json, text/plain, */*")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	return h
}

func isRateLimited(status int, body, loc string) bool {
	blob := strings.ToLower(body + "\n" + loc)
	if status == http.StatusTooManyRequests {
		return true
	}
	for _, needle := range []string{
		"rate_limited", "rate-limited", "too_many_requests", "too many",
		"ratelimit", "slow_down",
	} {
		if strings.Contains(blob, needle) {
			return true
		}
	}
	return false
}

func backoff(base float64, attempt int, capSec float64) time.Duration {
	if base <= 0 {
		base = 10
	}
	if attempt < 1 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 4 {
		shift = 4
	}
	d := base * float64(int(1)<<shift)
	if d > capSec {
		d = capSec
	}
	// small jitter 0-4s via nanosecond low bits
	jitter := time.Duration(time.Now().UnixNano()%5) * time.Second
	return time.Duration(d)*time.Second + jitter
}

func doRequest(client *http.Client, method, rawURL string, body io.Reader, headers http.Header) (status int, finalURL string, respBody []byte, err error) {
	req, errNew := http.NewRequest(method, rawURL, body)
	if errNew != nil {
		return 0, "", nil, errNew
	}
	for k, vals := range chromeHeaders() {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	for k, vals := range headers {
		req.Header.Del(k)
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, errDo := client.Do(req)
	if errDo != nil {
		return 0, "", nil, errDo
	}
	defer resp.Body.Close()
	data, errRead := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if errRead != nil {
		return resp.StatusCode, resp.Request.URL.String(), nil, errRead
	}
	if len(data) > 1<<20 {
		return resp.StatusCode, resp.Request.URL.String(), nil, fmt.Errorf("response body exceeds 1 MiB")
	}
	return resp.StatusCode, resp.Request.URL.String(), data, nil
}

func validateSSO(client *http.Client) error {
	status, finalURL, _, err := doRequest(client, http.MethodGet, xaiAccountsURL, nil, nil)
	if err != nil {
		return fmt.Errorf("validate sso: %w", err)
	}
	if status >= http.StatusBadRequest {
		return fmt.Errorf("validate sso HTTP %d", status)
	}
	low := strings.ToLower(finalURL)
	if strings.Contains(low, "sign-in") || strings.Contains(low, "sign-up") {
		return fmt.Errorf("sso cookie is invalid or expired")
	}
	return nil
}

func requestFreshDevice(client *http.Client, maxRetries int, baseDelay float64) (*deviceCodeResponse, error) {
	dc, err := requestDeviceCode(client, maxRetries, baseDelay)
	if err != nil {
		return nil, err
	}
	verificationURL := firstNonEmpty(dc.VerificationURIComplete, dc.VerificationURI)
	if verificationURL == "" {
		return nil, fmt.Errorf("device/code missing verification URI")
	}
	status, finalURL, body, err := doRequest(client, http.MethodGet, verificationURL, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("open verification URI: %w", err)
	}
	if isRateLimited(status, string(body), finalURL) {
		return nil, &rateLimitedError{msg: fmt.Sprintf("verification URI rate limited HTTP %d", status)}
	}
	if status >= http.StatusBadRequest {
		return nil, fmt.Errorf("open verification URI HTTP %d", status)
	}
	return dc, nil
}

func flowReached(finalURL, destination string) bool {
	u, err := url.Parse(strings.TrimSpace(finalURL))
	if err != nil {
		return false
	}
	want := "/" + strings.Trim(strings.ToLower(destination), "/")
	path := strings.ToLower(strings.TrimRight(u.Path, "/"))
	return path == want || strings.HasSuffix(path, want) || strings.Contains(path, want+"/")
}

func requestDeviceCode(client *http.Client, maxRetries int, baseDelay float64) (*deviceCodeResponse, error) {
	if maxRetries < 1 {
		maxRetries = 6
	}
	form := url.Values{
		"client_id": {xaiClientID},
		"scope":     {xaiScope},
	}
	var lastBody string
	for attempt := 1; attempt <= maxRetries; attempt++ {
		headers := make(http.Header)
		headers.Set("Content-Type", "application/x-www-form-urlencoded")
		status, _, body, err := doRequest(client, http.MethodPost, xaiDeviceCodeURL, strings.NewReader(form.Encode()), headers)
		if err != nil {
			if attempt < maxRetries {
				time.Sleep(backoff(baseDelay, attempt, 120))
				continue
			}
			return nil, fmt.Errorf("device/code: %w", err)
		}
		lastBody = string(body)
		if isRateLimited(status, lastBody, "") {
			if attempt < maxRetries {
				time.Sleep(backoff(baseDelay, attempt, 180))
				continue
			}
			return nil, &rateLimitedError{msg: fmt.Sprintf("device/code rate limited HTTP %d", status)}
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("device/code HTTP %d: %s", status, trimBody(lastBody, 200))
		}
		var dc deviceCodeResponse
		if errJSON := json.Unmarshal(body, &dc); errJSON != nil {
			return nil, fmt.Errorf("device/code parse: %w", errJSON)
		}
		if strings.TrimSpace(dc.DeviceCode) == "" || strings.TrimSpace(dc.UserCode) == "" {
			return nil, fmt.Errorf("device/code missing device_code/user_code")
		}
		return &dc, nil
	}
	return nil, &rateLimitedError{msg: "device/code retries exhausted: " + trimBody(lastBody, 80)}
}

func verifyAndApprove(client *http.Client, dc *deviceCodeResponse, maxRetries int, baseDelay float64) error {
	if maxRetries < 1 {
		maxRetries = 8
	}
	rateHits := 0
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// verify
		form := url.Values{"user_code": {dc.UserCode}}
		headers := make(http.Header)
		headers.Set("Content-Type", "application/x-www-form-urlencoded")
		status, finalURL, body, err := doRequest(client, http.MethodPost, xaiDeviceVerifyURL, strings.NewReader(form.Encode()), headers)
		if err != nil {
			time.Sleep(backoff(baseDelay, attempt, 120))
			continue
		}
		if isRateLimited(status, string(body), finalURL) {
			rateHits++
			time.Sleep(backoff(baseDelay, attempt, 180))
			// refresh device code
			fresh, errFresh := requestFreshDevice(client, maxRetries, baseDelay)
			if errFresh != nil {
				return errFresh
			}
			*dc = *fresh
			continue
		}
		if !flowReached(finalURL, "consent") {
			return fmt.Errorf("verify did not reach consent (HTTP %d): %s", status, finalURL)
		}

		// approve
		approveForm := url.Values{
			"user_code":      {dc.UserCode},
			"action":         {"allow"},
			"principal_type": {"User"},
			"principal_id":   {""},
		}
		status, finalURL, body, err = doRequest(client, http.MethodPost, xaiDeviceApproveURL, strings.NewReader(approveForm.Encode()), headers)
		if err != nil {
			time.Sleep(backoff(baseDelay, attempt, 120))
			continue
		}
		if isRateLimited(status, string(body), finalURL) {
			rateHits++
			time.Sleep(backoff(baseDelay, attempt, 180))
			fresh, errFresh := requestFreshDevice(client, maxRetries, baseDelay)
			if errFresh != nil {
				return errFresh
			}
			*dc = *fresh
			continue
		}
		if flowReached(finalURL, "done") {
			return nil
		}
		return fmt.Errorf("approve did not reach done (HTTP %d): %s", status, finalURL)
	}
	if rateHits > 0 {
		return &rateLimitedError{msg: "verify/approve rate limited"}
	}
	return fmt.Errorf("verify/approve retries exhausted")
}

func pollToken(client *http.Client, deviceCode string, intervalSec, expiresIn, timeoutSec int) (*TokenPayload, error) {
	if intervalSec <= 0 {
		intervalSec = 5
	}
	if timeoutSec <= 0 {
		timeoutSec = int(pollTimeout.Seconds())
	}
	if expiresIn <= 0 {
		expiresIn = 1800
	}
	deadline := time.Now().Add(time.Duration(min(expiresIn, timeoutSec)) * time.Second)
	interval := time.Duration(intervalSec) * time.Second
	for time.Now().Before(deadline) {
		time.Sleep(interval)
		form := url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"client_id":   {xaiClientID},
			"device_code": {deviceCode},
		}
		headers := make(http.Header)
		headers.Set("Content-Type", "application/x-www-form-urlencoded")
		status, _, body, err := doRequest(client, http.MethodPost, xaiTokenEndpoint, strings.NewReader(form.Encode()), headers)
		if err != nil {
			continue
		}
		if isRateLimited(status, string(body), "") && status == http.StatusTooManyRequests {
			return nil, &rateLimitedError{msg: "token endpoint rate limited"}
		}
		var payload struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
			AccessToken      string `json:"access_token"`
			RefreshToken     string `json:"refresh_token"`
			IDToken          string `json:"id_token"`
			TokenType        string `json:"token_type"`
			ExpiresIn        int    `json:"expires_in"`
		}
		if errJSON := json.Unmarshal(body, &payload); errJSON != nil {
			continue
		}
		if payload.Error != "" {
			switch payload.Error {
			case "authorization_pending":
				continue
			case "slow_down":
				interval += 5 * time.Second
				continue
			default:
				return nil, fmt.Errorf("token: %s %s", payload.Error, payload.ErrorDescription)
			}
		}
		if status != http.StatusOK || strings.TrimSpace(payload.AccessToken) == "" {
			return nil, fmt.Errorf("token HTTP %d: %s", status, trimBody(string(body), 160))
		}
		return &TokenPayload{
			AccessToken:  payload.AccessToken,
			RefreshToken: payload.RefreshToken,
			IDToken:      payload.IDToken,
			TokenType:    firstNonEmpty(payload.TokenType, "Bearer"),
			ExpiresIn:    payload.ExpiresIn,
		}, nil
	}
	return nil, fmt.Errorf("token poll timeout")
}

func fetchUserinfoEmail(client *http.Client, accessToken string) string {
	if strings.TrimSpace(accessToken) == "" {
		return ""
	}
	headers := make(http.Header)
	headers.Set("Authorization", "Bearer "+accessToken)
	headers.Set("Accept", "application/json")
	status, _, body, err := doRequest(client, http.MethodGet, xaiUserinfoURL, nil, headers)
	if err != nil || status != http.StatusOK {
		return ""
	}
	var info struct {
		Email string `json:"email"`
	}
	if errJSON := json.Unmarshal(body, &info); errJSON != nil {
		return ""
	}
	return strings.TrimSpace(info.Email)
}

func decodeJWTClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	seg := parts[1]
	if m := len(seg) % 4; m != 0 {
		seg += strings.Repeat("=", 4-m)
	}
	raw, err := base64.URLEncoding.DecodeString(seg)
	if err != nil {
		return nil
	}
	var claims map[string]any
	if errJSON := json.Unmarshal(raw, &claims); errJSON != nil {
		return nil
	}
	return claims
}

func claimString(claims map[string]any, key string) string {
	if claims == nil {
		return ""
	}
	if v, ok := claims[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func claimFloat(claims map[string]any, key string) (float64, bool) {
	if claims == nil {
		return 0, false
	}
	switch v := claims[key].(type) {
	case float64:
		return v, true
	case json.Number:
		f, err := v.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func rfc3339Sec(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339)
}

func credentialFileName(email, subject string) string {
	email = sanitizeFileSegment(email)
	if email != "" {
		return "xai-" + email + ".json"
	}
	subject = sanitizeFileSegment(subject)
	if subject != "" {
		return "xai-" + subject + ".json"
	}
	return fmt.Sprintf("xai-%d.json", time.Now().UnixMilli())
}

func sanitizeFileSegment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '@' || r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func tokenToAuthFile(token *TokenPayload, emailOverride string) (string, XAIAuthFile) {
	accessClaims := decodeJWTClaims(token.AccessToken)
	idClaims := decodeJWTClaims(token.IDToken)

	sub := firstNonEmpty(
		claimString(accessClaims, "sub"),
		claimString(accessClaims, "principal_id"),
		claimString(idClaims, "sub"),
		token.Subject,
	)
	email := firstNonEmpty(
		emailOverride,
		token.Email,
		claimString(idClaims, "email"),
		claimString(accessClaims, "email"),
	)

	expiresIn := token.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 21600
	}

	var expired string
	if exp, ok := claimFloat(accessClaims, "exp"); ok {
		expired = rfc3339Sec(time.Unix(int64(exp), 0))
	} else {
		expired = rfc3339Sec(time.Now().Add(time.Duration(expiresIn) * time.Second))
	}

	var lastRefresh string
	if iat, ok := claimFloat(accessClaims, "iat"); ok {
		lastRefresh = rfc3339Sec(time.Unix(int64(iat), 0))
	} else {
		lastRefresh = rfc3339Sec(time.Now())
	}

	auth := XAIAuthFile{
		Type:          "xai",
		AuthKind:      "oauth",
		AccessToken:   token.AccessToken,
		RefreshToken:  token.RefreshToken,
		IDToken:       token.IDToken,
		TokenType:     firstNonEmpty(token.TokenType, "Bearer"),
		ExpiresIn:     expiresIn,
		Expired:       expired,
		LastRefresh:   lastRefresh,
		Email:         email,
		Subject:       sub,
		BaseURL:       xaiDefaultAPIBase,
		TokenEndpoint: xaiTokenEndpoint,
		RedirectURI:   xaiRedirectURI,
		Disabled:      false,
	}
	return credentialFileName(email, sub), auth
}

// ConvertSSO runs the full SSO cookie → xAI OAuth auth-file conversion.
func ConvertSSO(opts ConvertOptions) (*ConvertResult, error) {
	sso := strings.TrimSpace(opts.SSO)
	if sso == "" {
		return nil, fmt.Errorf("sso cookie is required")
	}
	// Strip accidental "sso=" prefix
	if strings.HasPrefix(strings.ToLower(sso), "sso=") {
		sso = sso[4:]
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 8
	}
	baseDelay := opts.BaseDelaySec
	if baseDelay <= 0 {
		baseDelay = 15
	}
	pollTimeoutSec := opts.PollTimeoutSec
	if pollTimeoutSec <= 0 {
		pollTimeoutSec = 90
	}

	client := newHTTPClient()
	setSSOCookie(client, sso)

	if opts.ValidateSSO {
		if err := validateSSO(client); err != nil {
			return nil, err
		}
	}

	dc, errDC := requestFreshDevice(client, maxRetries, baseDelay)
	if errDC != nil {
		return nil, fmt.Errorf("device authorization: %w", errDC)
	}

	if err := verifyAndApprove(client, dc, maxRetries, baseDelay); err != nil {
		return nil, fmt.Errorf("authorize device: %w", err)
	}

	token, errTok := pollToken(client, dc.DeviceCode, dc.Interval, dc.ExpiresIn, pollTimeoutSec)
	if errTok != nil {
		return nil, fmt.Errorf("exchange token: %w", errTok)
	}

	email := strings.TrimSpace(opts.Email)
	if email == "" {
		email = fetchUserinfoEmail(client, token.AccessToken)
	}
	token.Email = email
	if token.Subject == "" {
		token.Subject = claimString(decodeJWTClaims(token.AccessToken), "sub")
	}

	name, auth := tokenToAuthFile(token, email)
	return &ConvertResult{
		FileName: name,
		Email:    auth.Email,
		Subject:  auth.Subject,
		Auth:     auth,
	}, nil
}

// ParseSSOList parses multi-line SSO input.
// Supports bare JWT lines or email----password----sso / email----sso.
func ParseSSOList(raw string) []struct {
	SSO   string
	Email string
} {
	var out []struct {
		SSO   string
		Email string
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		email := ""
		sso := line
		if strings.Contains(line, "----") {
			parts := make([]string, 0, 3)
			for _, p := range strings.Split(line, "----") {
				p = strings.TrimSpace(p)
				if p != "" {
					parts = append(parts, p)
				}
			}
			if len(parts) >= 2 {
				sso = parts[len(parts)-1]
				if strings.Contains(parts[0], "@") {
					email = parts[0]
				}
			}
		}
		out = append(out, struct {
			SSO   string
			Email string
		}{SSO: sso, Email: email})
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func trimBody(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func minFloat(values ...float64) float64 {
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func maxFloat(values ...float64) float64 {
	result := values[0]
	for _, value := range values[1:] {
		if value > result {
			result = value
		}
	}
	return result
}
