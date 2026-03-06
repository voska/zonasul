package vtex

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
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
