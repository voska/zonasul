package vtex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os/exec"
	"runtime"
)

type authStartResponse struct {
	AuthenticationToken         string `json:"authenticationToken"`
	ShowClassicAuthentication   bool   `json:"showClassicAuthentication"`
	ShowAccessKeyAuthentication bool   `json:"showAccessKeyAuthentication"`
}

func (c *Client) AuthStart() (*authStartResponse, error) {
	callbackURL := url.QueryEscape("https://www.zonasul.com.br/api/vtexid/oauth/finish")
	path := fmt.Sprintf("/api/vtexid/pub/authentication/start?scope=%s&callbackUrl=%s", AccountName, callbackURL)
	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("auth start: %w", err)
	}
	var resp authStartResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("auth start parse: %w", err)
	}
	return &resp, nil
}

func (c *Client) OAuthLoginURL(authToken string) string {
	return fmt.Sprintf("%s/api/vtexid/pub/authentication/oauth/redirect?authenticationToken=%s&providerName=%s",
		c.baseURL, authToken, url.QueryEscape("Cliente Zona Sul"))
}

func (c *Client) OAuthLogin() (string, error) {
	startResp, err := c.AuthStart()
	if err != nil {
		return "", err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		accountCookie := r.URL.Query().Get("accountAuthCookie")
		if accountCookie != "" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = fmt.Fprint(w, "<html><body><h2>Login successful!</h2><p>You can close this tab.</p></body></html>")
			tokenCh <- accountCookie
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, "<html><body><h2>Login failed</h2><p>No auth cookie received.</p></body></html>")
		errCh <- fmt.Errorf("no accountAuthCookie in callback")
	})

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(listener) }()
	defer func() { _ = srv.Shutdown(context.Background()) }()

	loginURL := c.OAuthLoginURL(startResp.AuthenticationToken)
	fmt.Printf("Opening browser for login...\n")
	fmt.Printf("If the browser doesn't open, visit:\n%s\n\n", loginURL)
	fmt.Printf("After login, you'll be redirected to complete authentication.\n")
	fmt.Printf("Local callback server listening on port %d\n", port)

	openBrowser(loginURL)

	select {
	case token := <-tokenCh:
		return token, nil
	case err := <-errCh:
		return "", err
	}
}

const (
	customAuthDomain  = "https://autenticacao.zonasul.com.br"
	oauthProviderName = "Cliente Zona Sul"
)

// CredentialLogin performs the full custom OAuth flow using email and password.
// It returns the VTEX JWT token (VtexIdclientAutCookie_zonasul).
func (c *Client) CredentialLogin(email, password string) (string, error) {
	// Step 1: Start VTEX auth to get an authentication token
	startResp, err := c.AuthStart()
	if err != nil {
		return "", err
	}

	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{
		Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 15 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Step 2: Follow the OAuth redirect to establish a session on the custom auth domain
	oauthURL := c.OAuthLoginURL(startResp.AuthenticationToken)
	resp, err := httpClient.Get(oauthURL)
	if err != nil {
		return "", fmt.Errorf("oauth redirect: %w", err)
	}
	_ = resp.Body.Close()

	// Step 3: POST credentials to the custom auth API
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	loginReq, err := http.NewRequest("POST", customAuthDomain+"/api/login", bytes.NewReader(loginPayload))
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.Header.Set("Accept", "application/json")
	loginReq.Header.Set("X-Requested-With", "XMLHttpRequest")

	loginResp, err := httpClient.Do(loginReq)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	defer func() { _ = loginResp.Body.Close() }()

	loginBody, _ := io.ReadAll(loginResp.Body)

	if loginResp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string              `json:"message"`
			Errors  map[string][]string `json:"errors"`
		}
		if json.Unmarshal(loginBody, &errResp) == nil && errResp.Message != "" {
			return "", fmt.Errorf("login failed: %s", errResp.Message)
		}
		return "", fmt.Errorf("login failed: HTTP %d", loginResp.StatusCode)
	}

	var loginResult struct {
		RedirectTo string `json:"redirectTo"`
	}
	if err := json.Unmarshal(loginBody, &loginResult); err != nil {
		return "", fmt.Errorf("login parse: %w", err)
	}
	if loginResult.RedirectTo == "" {
		return "", fmt.Errorf("login failed: no redirect URL in response")
	}

	// Step 4: Follow the OAuth authorize redirect chain to get the VTEX JWT
	// Stop redirecting so we can inspect the final callback URL
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		// Look for the accountAuthCookieValue in the redirect URL
		if v := req.URL.Query().Get("accountAuthCookieValue"); v != "" {
			return http.ErrUseLastResponse
		}
		if len(via) >= 15 {
			return fmt.Errorf("too many redirects")
		}
		return nil
	}

	finalResp, err := httpClient.Get(loginResult.RedirectTo)
	if err != nil && finalResp == nil {
		return "", fmt.Errorf("oauth callback: %w", err)
	}
	if finalResp != nil {
		_ = finalResp.Body.Close()
	}

	// Extract the JWT from the redirect chain
	// Check the Location header and the response URL for the token
	for finalResp != nil {
		location := finalResp.Header.Get("Location")
		if location != "" {
			parsedURL, parseErr := url.Parse(location)
			if parseErr == nil {
				if token := parsedURL.Query().Get("accountAuthCookieValue"); token != "" {
					return token, nil
				}
			}
		}

		// Follow one more redirect if needed
		if finalResp.StatusCode >= 300 && finalResp.StatusCode < 400 && location != "" {
			nextResp, nextErr := httpClient.Get(location)
			if nextErr != nil && nextResp == nil {
				return "", fmt.Errorf("following redirect: %w", nextErr)
			}
			if nextResp != nil {
				_ = nextResp.Body.Close()
			}
			finalResp = nextResp
			continue
		}
		break
	}

	return "", fmt.Errorf("login succeeded but failed to extract VTEX token from OAuth callback")
}

func (c *Client) AuthenticatedUser() (string, error) {
	body, err := c.Get("/api/vtexid/pub/authenticated/user")
	if err != nil {
		return "", err
	}
	var resp struct {
		User string `json:"user"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	if resp.User == "" {
		return "", fmt.Errorf("token expired or invalid")
	}
	return resp.User, nil
}

func openBrowser(url string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	}
}
