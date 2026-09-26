package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type consentTransport func(*http.Request) (*http.Response, error)

func (f consentTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestConsentConversionFlow(t *testing.T) {
	jwt := func(header, payload string) string {
		return base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + base64.RawURLEncoding.EncodeToString([]byte(payload)) + ".c2lnbmF0dXJl"
	}
	token := jwt(`{"alg":"ES256","typ":"consent+jwt"}`, `{"nonce":"first"}`)
	fresh := jwt(`{"typ":"consent+jwt","alg":"ES256"}`, `{"nonce":"second"}`)
	unrelated := jwt(`{"alg":"ES256","typ":"JWT"}`, `{}`)
	for _, tc := range []struct {
		name, page  string
		status      int
		retry, fail bool
	}{
		{name: "hidden input", page: `<input value="` + token + `" name="consent_token" type="hidden">`},
		{name: "serialized script", page: `<script>self.__next_f.push([1,"{\"consent_token\":\"` + token + `\"}"])</script>`},
		{name: "unrelated and repeated", page: unrelated + " " + token + " " + token},
		{name: "missing", page: unrelated, fail: true},
		{name: "malformed", page: "eyJ.invalid.signature", fail: true},
		{name: "wrong algorithm", page: jwt(`{"alg":"HS256","typ":"consent+jwt"}`, `{}`), fail: true},
		{name: "ambiguous", page: token + " " + fresh, fail: true},
		{name: "error page", page: token, status: 403, fail: true},
		{name: "fresh token after transport failure", page: token, retry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			verifies, approves, exchanges := 0, 0, 0
			old := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = old })
			http.DefaultTransport = consentTransport(func(r *http.Request) (*http.Response, error) {
				status, body := 200, ""
				headers := make(http.Header)
				redirect := func(path string) { status = 303; headers.Set("Location", "https://auth.x.ai"+path) }
				switch r.URL.Path {
				case "/oauth2/device/code":
					body = `{"device_code":"device","user_code":"user","verification_uri":"https://auth.x.ai/oauth2/device","interval":1,"expires_in":60}`
				case "/oauth2/device":
				case "/oauth2/device/verify":
					verifies++
					if err := r.ParseForm(); err != nil {
						t.Fatal(err)
					}
					if r.Form.Get("user_code") != "user" {
						t.Error("lost user code")
					}
					redirect("/oauth2/device/consent")
				case "/oauth2/device/consent":
					body = tc.page
					if tc.retry && verifies > 1 {
						body = fresh
					}
					if tc.status != 0 {
						status = tc.status
					}
				case "/oauth2/device/approve":
					approves++
					if err := r.ParseForm(); err != nil {
						t.Fatal(err)
					}
					want := token
					if verifies > 1 {
						want = fresh
					}
					if r.Form.Get("consent_token") != want {
						t.Error("approve missing current consent token")
					}
					if r.Form.Get("user_code") != "user" || r.Form.Get("action") != "allow" || r.Form.Get("principal_type") != "User" {
						t.Error("lost approval fields")
					}
					if c, err := r.Cookie("sso"); err != nil || c.Value != "test-sso" {
						t.Error("lost SSO cookie")
					}
					if tc.retry && approves == 1 {
						return nil, fmt.Errorf("simulated connection loss after token consumption")
					}
					redirect("/oauth2/device/done")
				case "/oauth2/device/done":
				case "/oauth2/token":
					exchanges++
					if err := r.ParseForm(); err != nil {
						t.Fatal(err)
					}
					if r.Form.Get("device_code") != "device" {
						t.Error("lost device code")
					}
					body = `{"access_token":"access","refresh_token":"refresh","expires_in":3600}`
				case "/oauth2/userinfo":
					body = `{"email":"test@example.com"}`
				default:
					t.Fatalf("unexpected path %s", r.URL.Path)
				}
				return &http.Response{StatusCode: status, Header: headers, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})
			result, err := ConvertSSO(ConvertOptions{SSO: "test-sso", MaxRetries: 2, Sleep: func(time.Duration) error { return nil }})
			if tc.fail {
				if err == nil || approves != 0 || exchanges != 0 {
					t.Fatalf("invalid consent was not stopped: error=%v approvals=%d exchanges=%d", err, approves, exchanges)
				}
				if strings.Contains(err.Error(), token) {
					t.Fatal("error leaked token")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Auth.AccessToken != "access" || result.Auth.RefreshToken != "refresh" || result.Auth.Email != "test@example.com" {
				t.Fatal("incomplete credentials")
			}
			want := 1
			if tc.retry {
				want = 2
			}
			if verifies != want || approves != want || exchanges != 1 {
				t.Fatalf("verify=%d approve=%d exchange=%d", verifies, approves, exchanges)
			}
		})
	}
}
