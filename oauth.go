package zonasul

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/voska/vtexkit/store"
)

const (
	// customAuthDomain is the Laravel app Zona Sul puts in front of VTEX ID.
	customAuthDomain = "https://autenticacao.zonasul.com.br"
	providerName     = "Cliente Zona Sul"
	maxRedirects     = 15
)

// Driver logs in through Zona Sul's custom OAuth provider.
//
// Classic VTEX password auth is disabled on this account, so credentials go
// to autenticacao.zonasul.com.br and the resulting OAuth redirect chain is
// followed back to a VTEX JWT. This is the only genuinely non-portable
// behavior across the stores surveyed, and it is the thing most likely to
// break if Zona Sul changes their auth app.
type Driver struct{}

func (Driver) ProviderName() string { return providerName }

func (Driver) Login(ctx context.Context, _ store.HTTPDoer, baseURL, email, password string) (string, error) {
	// A dedicated jar and redirect policy are needed here: the chain sets
	// cookies across two domains and must be halted at the callback to read
	// the token out of the redirect URL rather than following it.
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	authToken, err := startAuth(ctx, client, baseURL)
	if err != nil {
		return "", err
	}

	// Establish a session on the custom auth domain.
	oauthURL := fmt.Sprintf(
		"%s/api/vtexid/pub/authentication/oauth/redirect?authenticationToken=%s&providerName=%s",
		baseURL, authToken, url.QueryEscape(providerName))
	resp, err := ctxGet(ctx, client, oauthURL)
	if err != nil {
		return "", fmt.Errorf("oauth redirect: %w", err)
	}
	_ = resp.Body.Close()

	redirectTo, err := submitCredentials(ctx, client, email, password)
	if err != nil {
		return "", err
	}

	return followToToken(ctx, client, redirectTo)
}

func startAuth(ctx context.Context, client *http.Client, baseURL string) (string, error) {
	callback := url.QueryEscape(baseURL + "/api/vtexid/oauth/finish")
	u := fmt.Sprintf("%s/api/vtexid/pub/authentication/start?scope=zonasul&callbackUrl=%s", baseURL, callback)
	resp, err := ctxGet(ctx, client, u)
	if err != nil {
		return "", fmt.Errorf("auth start: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var parsed struct {
		AuthenticationToken string `json:"authenticationToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("auth start parse: %w", err)
	}
	if parsed.AuthenticationToken == "" {
		return "", fmt.Errorf("auth start returned no authentication token")
	}
	return parsed.AuthenticationToken, nil
}

func submitCredentials(ctx context.Context, client *http.Client, email, password string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		customAuthDomain+"/api/login", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && errResp.Message != "" {
			return "", fmt.Errorf("login failed: %s", errResp.Message)
		}
		return "", fmt.Errorf("login failed: HTTP %d", resp.StatusCode)
	}

	var result struct {
		RedirectTo string `json:"redirectTo"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("login parse: %w", err)
	}
	if result.RedirectTo == "" {
		return "", fmt.Errorf("login succeeded but returned no redirect URL")
	}
	return result.RedirectTo, nil
}

// followToToken walks the OAuth callback chain, stopping as soon as a
// redirect carries the VTEX JWT as a query parameter.
func followToToken(ctx context.Context, client *http.Client, startURL string) (string, error) {
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Query().Get("accountAuthCookieValue") != "" {
			return http.ErrUseLastResponse
		}
		if len(via) >= maxRedirects {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}

	next := startURL
	for range maxRedirects {
		resp, err := ctxGet(ctx, client, next)
		if err != nil {
			return "", fmt.Errorf("oauth callback: %w", err)
		}
		location := resp.Header.Get("Location")
		status := resp.StatusCode
		_ = resp.Body.Close()

		if location != "" {
			if parsed, parseErr := url.Parse(location); parseErr == nil {
				if token := parsed.Query().Get("accountAuthCookieValue"); token != "" {
					return token, nil
				}
			}
		}
		if status < 300 || status >= 400 || location == "" {
			break
		}
		next = location
	}
	return "", fmt.Errorf("login succeeded but no VTEX token appeared in the OAuth callback")
}

func ctxGet(ctx context.Context, client *http.Client, u string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}
