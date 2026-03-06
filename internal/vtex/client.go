package vtex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
)

const (
	AccountName = "zonasul"
	SellerID    = "zonasulzsa"
	BindingID   = "0a362f40-93e7-42c4-90c5-a3946de77fb3"
	BaseURL     = "https://www.zonasul.com.br"
)

type Client struct {
	baseURL       string
	authToken     string
	httpClient    *http.Client
	GatewayURL    string // Override for testing; empty = use production URL
}

func NewClient(baseURL, authToken string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL:    baseURL,
		authToken:  authToken,
		httpClient: &http.Client{Jar: jar},
	}
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	if c.authToken != "" {
		req.AddCookie(&http.Cookie{
			Name:  "VtexIdclientAutCookie_" + AccountName,
			Value: c.authToken,
		})
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return body, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) Get(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func (c *Client) PostJSON(path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) PatchJSON(path string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("PATCH", c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req)
}

func (c *Client) PostJSONAbsolute(absoluteURL string, payload any) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", absoluteURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", BaseURL)
	req.Header.Set("Referer", BaseURL+"/")
	if c.authToken != "" {
		req.AddCookie(&http.Cookie{
			Name:  "VtexIdclientAutCookie_" + AccountName,
			Value: c.authToken,
		})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return body, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}
