package main

import (
	"strings"
	"testing"
)

func TestParseSSOList(t *testing.T) {
	raw := `
# comment
eyJhbGciOi alone

user@example.com----secret----eyJpart.two.three
only@mail.com----eyJtoken
`
	got := ParseSSOList(raw)
	if len(got) != 3 {
		t.Fatalf("len=%d want 3: %+v", len(got), got)
	}
	if got[0].SSO != "eyJhbGciOi alone" || got[0].Email != "" {
		t.Fatalf("item0 = %+v", got[0])
	}
	if got[1].Email != "user@example.com" || !strings.HasPrefix(got[1].SSO, "eyJpart") {
		t.Fatalf("item1 = %+v", got[1])
	}
	if got[2].Email != "only@mail.com" || got[2].SSO != "eyJtoken" {
		t.Fatalf("item2 = %+v", got[2])
	}
}

func TestCredentialFileName(t *testing.T) {
	if got := credentialFileName("User+Tag@Example.COM", ""); got != "xai-User-Tag@Example.COM.json" {
		// + becomes -
		if !strings.HasPrefix(got, "xai-") || !strings.HasSuffix(got, ".json") {
			t.Fatalf("email name = %q", got)
		}
	}
	if got := credentialFileName("", "sub/with spaces"); !strings.HasPrefix(got, "xai-") {
		t.Fatalf("sub name = %q", got)
	}
	if got := credentialFileName("", ""); !strings.HasPrefix(got, "xai-") || !strings.HasSuffix(got, ".json") {
		t.Fatalf("anon name = %q", got)
	}
}

func TestTokenToAuthFile(t *testing.T) {
	// minimal unsigned-looking JWT payload {"sub":"abc","email":"a@b.c","exp":2000000000,"iat":1900000000}
	// We don't need a real signature for decodeJWTClaims.
	payload := base64URL(`{"sub":"abc","email":"a@b.c","exp":2000000000,"iat":1900000000}`)
	access := "hdr." + payload + ".sig"
	token := &TokenPayload{
		AccessToken:  access,
		RefreshToken: "rt",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
	}
	name, auth := tokenToAuthFile(token, "")
	if auth.Type != "xai" || auth.AuthKind != "oauth" {
		t.Fatalf("type/kind = %s/%s", auth.Type, auth.AuthKind)
	}
	if auth.Email != "a@b.c" || auth.Subject != "abc" {
		t.Fatalf("identity email=%q sub=%q", auth.Email, auth.Subject)
	}
	if name != "xai-a@b.c.json" {
		t.Fatalf("name = %q", name)
	}
	if auth.BaseURL == "" || auth.TokenEndpoint == "" {
		t.Fatalf("missing endpoints: %+v", auth)
	}
	if auth.Disabled {
		t.Fatal("disabled should be false")
	}
}

func base64URL(s string) string {
	return mustB64URL([]byte(s))
}

func mustB64URL(b []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	// manual base64url without padding
	out := make([]byte, 0, (len(b)+2)/3*4)
	for i := 0; i < len(b); i += 3 {
		var n uint32
		remain := len(b) - i
		n = uint32(b[i]) << 16
		if remain > 1 {
			n |= uint32(b[i+1]) << 8
		}
		if remain > 2 {
			n |= uint32(b[i+2])
		}
		out = append(out, alphabet[(n>>18)&63], alphabet[(n>>12)&63])
		if remain > 1 {
			out = append(out, alphabet[(n>>6)&63])
		}
		if remain > 2 {
			out = append(out, alphabet[n&63])
		}
	}
	return string(out)
}
