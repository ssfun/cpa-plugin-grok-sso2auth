# Consent flow regression

Failure scenarios fixed before implementation:

- Consent token in an HTML input or serialized script is omitted from approve.
- An unrelated JWT is mistaken for the ES256 consent+jwt token.
- Missing, malformed, wrong-algorithm, or conflicting consent tokens trigger approve.
- An HTTP error at the consent URL is treated as a valid page.
- An approve transport failure reuses a consumed token instead of verifying again.
- A successful flow loses the device code, cookies, or final OAuth credentials.
- Approve includes a valid consent token but omits the accounts.x.ai Origin,
  causing HTTP 403 "Request could not be verified" (confirmed with three live accounts).
- Origin is lost on retry or is incorrectly set to the auth.x.ai API host.
- The fixture keeps consent on auth.x.ai and misses the real cross-host redirect.

Run the mocked-provider conversion flow and existing checks, saving repeatable evidence:

```sh
mkdir -p dist/validation
go test -json ./... > dist/validation/go-test.json
go vet ./...
```

The regression executes ConvertSSO through HTTP redirects, approval, token exchange,
and userinfo using an in-memory xAI transport. It does not validate the live xAI
page or JWT signatures; signature verification belongs to the xAI approve endpoint.
