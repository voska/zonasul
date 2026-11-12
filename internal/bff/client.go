// Package bff implements the current Zona Sul storefront API.
package bff

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/voska/vtexkit/cli/errfmt"
)

const (
	BaseURL  = "https://www.zonasul.com.br/api/bff"
	Contract = "E9A2FCA6-198F-4B4A-A183-2356EF6562F5"
)

type Client struct {
	base string
	http *http.Client
}

func New(base string, transport *http.Client) *Client {
	if transport == nil {
		transport = &http.Client{Timeout: 30 * time.Second}
	}
	client := *transport
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{base: strings.TrimRight(base, "/"), http: &client}
}

// Request rejects failed envelopes, redirects and missing data. Server bodies
// and transport errors can contain account data, so errors carry status only.
func (c *Client) Request(method, path, token string, payload, result any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return errfmt.Usage("invalid BFF request")
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return errfmt.Config("invalid BFF URL")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-contractaccountid", Contract)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return errfmt.Retryable("BFF request did not complete")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		switch resp.StatusCode {
		case 401:
			return errfmt.Auth("BFF authentication rejected; run zonasul auth login or auth code send")
		case 403:
			return errfmt.Forbidden("BFF access denied")
		case 404:
			return errfmt.NotFound("BFF resource not found")
		case 429:
			return errfmt.RateLimit()
		default:
			code := errfmt.ExitDomain
			if resp.StatusCode >= 500 {
				code = errfmt.ExitRetryable
			}
			return errfmt.New(code, fmt.Sprintf("BFF returned HTTP %d", resp.StatusCode))
		}
	}
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(&envelope) != nil {
		return errfmt.Domain("BFF returned invalid JSON")
	}
	if !envelope.Success {
		return errfmt.Domain("BFF did not confirm success")
	}
	if result != nil && (len(envelope.Data) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) || json.Unmarshal(envelope.Data, result) != nil) {
		return errfmt.Domain("BFF returned missing or invalid data")
	}
	return nil
}

type Identity struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func (c *Client) Identity(token, email string) (*Identity, error) {
	if token == "" || strings.TrimSpace(email) == "" {
		return nil, errfmt.Auth("BFF token and expected email are required")
	}
	var identity Identity
	if err := c.Request(http.MethodGet, "/users/me", token, nil, &identity); err != nil {
		return nil, err
	}
	if identity.ID == "" || !strings.EqualFold(identity.Email, email) {
		return nil, errfmt.Auth("BFF identity does not match the requested account")
	}
	return &identity, nil
}

func (c *Client) login(path, email string, payload any) (string, error) {
	var result struct {
		Token string `json:"token"`
	}
	if err := c.Request(http.MethodPost, path, "", payload, &result); err != nil {
		var typed *errfmt.Error
		if errors.As(err, &typed) && typed.Code == errfmt.ExitDomain {
			return "", errfmt.Auth("BFF login rejected: " + typed.Message)
		}
		return "", err
	}
	if _, err := c.Identity(result.Token, email); err != nil {
		return "", err
	}
	return result.Token, nil
}

func (c *Client) Login(email, password string) (string, error) {
	return c.login("/login", email, map[string]string{"login": email, "password": password})
}
func (c *Client) SendCode(email string) (string, error) {
	var result struct {
		UserID string `json:"userId"`
	}
	if err := c.Request(http.MethodPost, "/login/email/send-code", "", map[string]string{"email": email}, &result); err != nil {
		return "", err
	}
	if result.UserID == "" {
		return "", errfmt.Auth("BFF did not return a login userId")
	}
	return result.UserID, nil
}
func (c *Client) VerifyCode(email, userID, code string) (string, error) {
	return c.login("/login/email", email, map[string]string{"userId": userID, "code": code})
}
