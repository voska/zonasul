# Zona Sul CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI (`zonasul`) that lets an AI agent search products, manage a cart, and place grocery orders on zonasul.com.br via their VTEX API.

**Architecture:** Single Go binary using Kong CLI framework. VTEX API client in `internal/vtex/` handles HTTP, auth cookies, GraphQL persisted queries, and REST checkout. Config + keychain for credential persistence. Triple output mode (human/json/plain) for agent compatibility.

**Tech Stack:** Go 1.22+, Kong (CLI), go-keyring (macOS Keychain), net/http (VTEX API)

**Reference:** All API endpoints, hashes, and response shapes in `docs/zonasul-api-research.md`

---

### Task 1: Project Scaffold

**Files:**
- Create: `go.mod`
- Create: `cmd/zonasul/main.go`
- Create: `internal/exitcode/exitcode.go`

**Step 1: Initialize Go module**

```bash
cd /path/to/project
go mod init github.com/mattvoska/zonasul
```

**Step 2: Install Kong**

```bash
go get github.com/alecthomas/kong
```

**Step 3: Write exit codes**

Create `internal/exitcode/exitcode.go`:
```go
package exitcode

const (
	Success      = 0
	Error        = 1
	Usage        = 2
	EmptyResults = 3
	AuthRequired = 4
	NotFound     = 5
	MinOrder     = 6
	RateLimited  = 7
)
```

**Step 4: Write CLI skeleton with Kong**

Create `cmd/zonasul/main.go`:
```go
package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/mattvoska/zonasul/internal/exitcode"
)

type Globals struct {
	JSON    bool `help:"Output JSON for agent consumption." env:"ZONASUL_JSON"`
	Plain   bool `help:"Output plain text for piping." env:"ZONASUL_PLAIN"`
	NoInput bool `help:"Disable interactive prompts." env:"ZONASUL_NO_INPUT"`
}

type AuthLoginCmd struct{}
type AuthStatusCmd struct{}
type AuthLogoutCmd struct{}

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Login with email and password."`
	Status AuthStatusCmd `cmd:"" help:"Check auth state and token expiry."`
	Logout AuthLogoutCmd `cmd:"" help:"Clear stored credentials."`
}

type SearchCmd struct {
	Query string `arg:"" help:"Search query."`
	Limit int    `help:"Max results." default:"20"`
}

type CartAddCmd struct {
	SKU string `arg:"" help:"SKU ID to add."`
	Qty int    `help:"Quantity." default:"1"`
}

type CartRemoveCmd struct {
	Index int `arg:"" help:"Cart item index to remove."`
}

type CartClearCmd struct{}

type CartCmd struct {
	Add    CartAddCmd    `cmd:"" help:"Add item to cart."`
	Remove CartRemoveCmd `cmd:"" help:"Remove item from cart."`
	Clear  CartClearCmd  `cmd:"" help:"Clear all items from cart."`
}

type DeliveryWindowsCmd struct{}

type DeliveryCmd struct {
	Windows DeliveryWindowsCmd `cmd:"" help:"List available delivery windows."`
}

type CheckoutCmd struct {
	Window  int  `help:"Delivery window index." default:"-1"`
	Payment int  `help:"Payment method ID." default:"-1"`
	Confirm bool `help:"Actually place the order. Required safety gate."`
}

type CLI struct {
	Globals

	Auth     AuthCmd     `cmd:"" help:"Authentication commands."`
	Search   SearchCmd   `cmd:"" help:"Search products."`
	Cart     CartCmd     `cmd:"" help:"Manage shopping cart."`
	Delivery DeliveryCmd `cmd:"" help:"Delivery options."`
	Checkout CheckoutCmd `cmd:"" help:"Place an order."`
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("zonasul"),
		kong.Description("Zona Sul supermarket CLI for AI agents."),
		kong.UsageOnError(),
	)

	err := ctx.Run(&cli.Globals)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.Error)
	}
}

func (c *AuthLoginCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "auth login: not implemented")
	return nil
}

func (c *AuthStatusCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "auth status: not implemented")
	return nil
}

func (c *AuthLogoutCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "auth logout: not implemented")
	return nil
}

func (c *SearchCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "search: not implemented")
	return nil
}

func (c *CartAddCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "cart add: not implemented")
	return nil
}

func (c *CartRemoveCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "cart remove: not implemented")
	return nil
}

func (c *CartClearCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "cart clear: not implemented")
	return nil
}

func (c *DeliveryWindowsCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "delivery windows: not implemented")
	return nil
}

func (c *CheckoutCmd) Run(globals *Globals) error {
	fmt.Fprintln(os.Stderr, "checkout: not implemented")
	return nil
}
```

**Step 5: Build and verify**

```bash
go build -o zonasul ./cmd/zonasul
./zonasul --help
./zonasul search banana
```

Expected: help text shows all commands; `search banana` prints "search: not implemented" to stderr.

**Step 6: Commit**

```bash
git init
git add go.mod go.sum cmd/ internal/ docs/ CLAUDE.md .gitignore
git commit -m "feat: project scaffold with Kong CLI and exit codes"
```

---

### Task 2: Output Formatting

**Files:**
- Create: `internal/outfmt/outfmt.go`
- Create: `internal/outfmt/outfmt_test.go`

**Step 1: Write the test**

Create `internal/outfmt/outfmt_test.go`:
```go
package outfmt_test

import (
	"bytes"
	"testing"

	"github.com/mattvoska/zonasul/internal/outfmt"
)

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	f := outfmt.New(true, false, &buf)
	data := map[string]string{"name": "Banana"}
	if err := f.Print(data); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if got != "{\"name\":\"Banana\"}\n" {
		t.Errorf("got %q", got)
	}
}

func TestHuman(t *testing.T) {
	var buf bytes.Buffer
	f := outfmt.New(false, false, &buf)
	if err := f.Print("hello"); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "hello\n" {
		t.Errorf("got %q", buf.String())
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/outfmt/ -v
```

Expected: FAIL (package doesn't exist)

**Step 3: Write implementation**

Create `internal/outfmt/outfmt.go`:
```go
package outfmt

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Formatter struct {
	json   bool
	plain  bool
	writer io.Writer
}

func New(jsonMode, plainMode bool, w io.Writer) *Formatter {
	return &Formatter{json: jsonMode, plain: plainMode, writer: w}
}

func FromGlobals(jsonMode, plainMode bool) *Formatter {
	return New(jsonMode, plainMode, os.Stdout)
}

func (f *Formatter) Print(v any) error {
	if f.json {
		enc := json.NewEncoder(f.writer)
		return enc.Encode(v)
	}
	_, err := fmt.Fprintln(f.writer, v)
	return err
}

func (f *Formatter) IsJSON() bool { return f.json }
```

**Step 4: Run test to verify it passes**

```bash
go test ./internal/outfmt/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/outfmt/
git commit -m "feat: triple output formatter (json/plain/human)"
```

---

### Task 3: Config Management

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write the test**

Create `internal/config/config_test.go`:
```go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattvoska/zonasul/internal/config"
)

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &config.Config{
		CEP:          "22240-003",
		Street:       "Rua das Laranjeiras",
		Number:       "100",
		Complement:   "Apto 200",
		Neighborhood: "Laranjeiras",
		City:         "Rio de Janeiro",
		State:        "RJ",
		OrderFormID:  "abc123",
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.CEP != "22240-003" || loaded.Number != "100" || loaded.OrderFormID != "abc123" {
		t.Errorf("config mismatch: %+v", loaded)
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path.json")
	if err != nil {
		t.Fatal("should not error on missing file")
	}
	if cfg.CEP != "" {
		t.Error("expected empty config")
	}
	_ = cfg
}

func TestDefaultPath(t *testing.T) {
	p := config.DefaultPath()
	if !filepath.IsAbs(p) {
		t.Errorf("expected absolute path, got %s", p)
	}
	_ = os.Getenv("HOME") // just ensure it resolves
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/config/ -v
```

**Step 3: Write implementation**

Create `internal/config/config.go`:
```go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	CEP          string `json:"cep,omitempty"`
	Street       string `json:"street,omitempty"`
	Number       string `json:"number,omitempty"`
	Complement   string `json:"complement,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	State        string `json:"state,omitempty"`
	OrderFormID  string `json:"orderFormId,omitempty"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zonasul", "config.json")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
```

**Step 4: Run tests**

```bash
go test ./internal/config/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config load/save with JSON file at ~/.config/zonasul/"
```

---

### Task 4: VTEX HTTP Client Foundation

**Files:**
- Create: `internal/vtex/client.go`
- Create: `internal/vtex/client_test.go`

**Step 1: Write the test**

Create `internal/vtex/client_test.go`:
```go
package vtex_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattvoska/zonasul/internal/vtex"
)

func TestClientGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/test" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		cookie, err := r.Cookie("VtexIdclientAutCookie_zonasul")
		if err != nil || cookie.Value != "test-jwt" {
			t.Error("missing auth cookie")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	body, err := c.Get("/api/test")
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("got %q", string(body))
	}
}

func TestClientPostJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("missing content-type")
		}
		w.Write([]byte(`{"created":true}`))
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "test-jwt")
	body, err := c.PostJSON("/api/test", map[string]string{"key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"created":true}` {
		t.Errorf("got %q", string(body))
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/vtex/ -v
```

**Step 3: Write implementation**

Create `internal/vtex/client.go`:
```go
package vtex

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	AccountName = "zonasul"
	SellerID    = "zonasulzsa"
	BindingID   = "0a362f40-93e7-42c4-90c5-a3946de77fb3"
	BaseURL     = "https://www.zonasul.com.br"
)

type Client struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

func NewClient(baseURL, authToken string) *Client {
	return &Client{
		baseURL:    baseURL,
		authToken:  authToken,
		httpClient: &http.Client{},
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
	defer resp.Body.Close()

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

func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}
```

**Step 4: Run tests**

```bash
go test ./internal/vtex/ -v
```

Expected: PASS

**Step 5: Commit**

```bash
git add internal/vtex/
git commit -m "feat: VTEX HTTP client with cookie auth and JSON support"
```

---

### Task 5: Authentication (Login + Keychain)

**Files:**
- Create: `internal/vtex/auth.go`
- Create: `internal/vtex/auth_test.go`
- Modify: `cmd/zonasul/main.go` (wire up auth commands)

**Step 1: Install go-keyring**

```bash
go get github.com/zalando/go-keyring
```

**Step 2: Write the test**

Create `internal/vtex/auth_test.go`:
```go
package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattvoska/zonasul/internal/vtex"
)

func TestAuthStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/vtexid/pub/authentication/start" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		scope := r.URL.Query().Get("scope")
		if scope != "zonasul" {
			t.Errorf("expected scope=zonasul, got %s", scope)
		}
		json.NewEncoder(w).Encode(map[string]string{
			"authenticationToken": "test-auth-token-123",
		})
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	token, err := c.AuthStart()
	if err != nil {
		t.Fatal(err)
	}
	if token != "test-auth-token-123" {
		t.Errorf("got %q", token)
	}
}

func TestAuthClassicValidate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["login"] != "test@example.com" || body["password"] != "secret" {
			t.Error("wrong credentials")
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "VtexIdclientAutCookie_zonasul",
			Value: "jwt-token-abc",
		})
		json.NewEncoder(w).Encode(map[string]string{
			"authStatus": "Success",
		})
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	jwt, err := c.AuthClassicValidate("test-auth-token", "test@example.com", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if jwt != "jwt-token-abc" {
		t.Errorf("got %q", jwt)
	}
}
```

**Step 3: Run test to verify it fails**

```bash
go test ./internal/vtex/ -v -run TestAuth
```

**Step 4: Write implementation**

Create `internal/vtex/auth.go`:
```go
package vtex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type authStartResponse struct {
	AuthenticationToken string `json:"authenticationToken"`
}

func (c *Client) AuthStart() (string, error) {
	path := "/api/vtexid/pub/authentication/start?scope=" + AccountName
	body, err := c.Get(path)
	if err != nil {
		return "", fmt.Errorf("auth start: %w", err)
	}
	var resp authStartResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("auth start parse: %w", err)
	}
	return resp.AuthenticationToken, nil
}

func (c *Client) AuthClassicValidate(authToken, email, password string) (string, error) {
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{Jar: jar}

	payload := map[string]string{
		"authenticationToken": authToken,
		"login":              email,
		"password":           password,
	}
	data, _ := json.Marshal(payload)

	reqURL := c.baseURL + "/api/vtexid/pub/authentication/classic/validate"
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth validate: %w", err)
	}
	defer resp.Body.Close()

	cookieName := "VtexIdclientAutCookie_" + AccountName
	u, _ := url.Parse(c.baseURL)
	for _, cookie := range jar.Cookies(u) {
		if cookie.Name == cookieName {
			return cookie.Value, nil
		}
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == cookieName {
			return cookie.Value, nil
		}
	}

	return "", fmt.Errorf("auth validate: no %s cookie in response", cookieName)
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
```

Note: needs `"bytes"` import added to the import block.

**Step 5: Run tests**

```bash
go test ./internal/vtex/ -v -run TestAuth
```

Expected: PASS

**Step 6: Wire up auth commands in main.go**

Update `cmd/zonasul/main.go` auth command `Run` methods to call the VTEX client with interactive prompts for email/password, store JWT via `go-keyring`, and load from keyring on subsequent runs. (Full wiring code deferred to implementation - the pattern is: read config, create client, call auth methods, store token in keyring.)

**Step 7: Commit**

```bash
git add internal/vtex/auth.go internal/vtex/auth_test.go go.sum
git commit -m "feat: VTEX auth start + classic validate + keychain storage"
```

---

### Task 6: Product Search

**Files:**
- Create: `internal/vtex/search.go`
- Create: `internal/vtex/search_test.go`

**Step 1: Write the test**

Create `internal/vtex/search_test.go`:
```go
package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattvoska/zonasul/internal/vtex"
)

func TestSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		op := r.URL.Query().Get("operationName")
		if op != "productSearchV3" {
			t.Errorf("expected productSearchV3, got %s", op)
		}
		resp := map[string]any{
			"data": map[string]any{
				"productSearch": map[string]any{
					"products": []map[string]any{
						{
							"productId":   "6196",
							"productName": "Banana Prata Orgânica 800g",
							"items": []map[string]any{
								{
									"itemId": "6180",
									"name":   "Banana Prata Orgânica 800g",
									"sellers": []map[string]any{
										{
											"sellerId": "1",
											"commertialOffer": map[string]any{
												"Price":             10.99,
												"ListPrice":         10.99,
												"AvailableQuantity": 99999,
											},
										},
									},
									"measurementUnit": "kg",
									"unitMultiplier":  0.8,
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := vtex.NewClient(srv.URL, "")
	results, err := c.Search("banana", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].SKU != "6180" || results[0].Name != "Banana Prata Orgânica 800g" {
		t.Errorf("unexpected result: %+v", results[0])
	}
	if results[0].Price != 10.99 {
		t.Errorf("expected price 10.99, got %f", results[0].Price)
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/vtex/ -v -run TestSearch
```

**Step 3: Write implementation**

Create `internal/vtex/search.go`:
```go
package vtex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
)

type SearchResult struct {
	ProductID   string  `json:"productId"`
	SKU         string  `json:"sku"`
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	ListPrice   float64 `json:"listPrice"`
	Available   int     `json:"available"`
	Unit        string  `json:"unit"`
	UnitMult    float64 `json:"unitMultiplier"`
}

const searchHash = "31d3fa494df1fc41efef6d16dd96a96e6911b8aed7a037868699a1f3f4d365de"

func (c *Client) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	vars := map[string]any{
		"hideUnavailableItems":  true,
		"skusFilter":           "ALL",
		"simulationBehavior":   "default",
		"installmentCriteria":  "MAX_WITHOUT_INTEREST",
		"productOriginVtex":    true,
		"map":                  "ft",
		"query":                query,
		"orderBy":              "OrderByScoreDESC",
		"from":                 0,
		"to":                   limit - 1,
		"selectedFacets":       []map[string]string{{"key": "ft", "value": query}},
		"fullText":             query,
		"facetsBehavior":       "Static",
		"categoryTreeBehavior": "default",
		"withFacets":           false,
		"variant":              "null-null",
	}

	varsJSON, _ := json.Marshal(vars)
	varsB64 := base64.StdEncoding.EncodeToString(varsJSON)

	extensions := map[string]any{
		"persistedQuery": map[string]any{
			"version":    1,
			"sha256Hash": searchHash,
			"sender":     "vtex.store-resources@0.x",
			"provider":   "vtex.search-graphql@0.x",
		},
		"variables": varsB64,
	}
	extJSON, _ := json.Marshal(extensions)

	path := fmt.Sprintf("/_v/segment/graphql/v1?workspace=master&maxAge=short&appsEtag=remove&domain=store&locale=pt-BR&__bindingId=%s&operationName=productSearchV3&variables=%%7B%%7D&extensions=%s",
		BindingID, url.QueryEscape(string(extJSON)))

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	var resp struct {
		Data struct {
			ProductSearch struct {
				Products []struct {
					ProductID   string `json:"productId"`
					ProductName string `json:"productName"`
					Items       []struct {
						ItemID  string `json:"itemId"`
						Name    string `json:"name"`
						Sellers []struct {
							SellerID        string `json:"sellerId"`
							CommertialOffer struct {
								Price             float64 `json:"Price"`
								ListPrice         float64 `json:"ListPrice"`
								AvailableQuantity int     `json:"AvailableQuantity"`
							} `json:"commertialOffer"`
						} `json:"sellers"`
						MeasurementUnit string  `json:"measurementUnit"`
						UnitMultiplier  float64 `json:"unitMultiplier"`
					} `json:"items"`
				} `json:"products"`
			} `json:"productSearch"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("search parse: %w", err)
	}

	var results []SearchResult
	for _, p := range resp.Data.ProductSearch.Products {
		for _, item := range p.Items {
			r := SearchResult{
				ProductID: p.ProductID,
				SKU:       item.ItemID,
				Name:      item.Name,
				Unit:      item.MeasurementUnit,
				UnitMult:  item.UnitMultiplier,
			}
			if len(item.Sellers) > 0 {
				r.Price = item.Sellers[0].CommertialOffer.Price
				r.ListPrice = item.Sellers[0].CommertialOffer.ListPrice
				r.Available = item.Sellers[0].CommertialOffer.AvailableQuantity
			}
			results = append(results, r)
		}
	}
	return results, nil
}
```

**Step 4: Run tests**

```bash
go test ./internal/vtex/ -v -run TestSearch
```

Expected: PASS

**Step 5: Wire up search command in main.go**

Update `SearchCmd.Run` to create a client, call `Search`, and format output via `outfmt.Formatter`.

**Step 6: Commit**

```bash
git add internal/vtex/search.go internal/vtex/search_test.go
git commit -m "feat: product search via VTEX persisted GraphQL query"
```

---

### Task 7: Cart Management (Add / Remove / Clear / Show)

**Files:**
- Create: `internal/vtex/cart.go`
- Create: `internal/vtex/cart_test.go`

**Step 1: Write the test**

Test `AddToCart` using a mock server that expects the GraphQL mutation with correct hash and returns an orderForm response. Test `GetOrderForm` with a mock that returns items, address, and delivery windows. Test `RemoveItems` and `UpdateItems`.

**Step 2: Write implementation**

`internal/vtex/cart.go` implements:
- `AddToCart(skuID string, quantity int) (*OrderForm, error)` - persisted GraphQL mutation (hash `a63161354718146c4282079551df81aaa8fa3d59584520cf5ea1c278fac0db33`)
- `GetOrderForm() (*OrderForm, error)` - `GET /api/checkout/pub/orderForm`
- `UpdateItemQuantity(orderFormID string, index, quantity int) (*OrderForm, error)` - `POST /api/checkout/pub/orderForm/{id}/items/update`
- `RemoveAllItems(orderFormID string) error` - `POST /api/checkout/pub/orderForm/{id}/items/removeAll`

`OrderForm` struct contains: OrderFormID, Items, Totalizers, ShippingData (with address + delivery windows), PaymentData (available payment systems).

**Step 3: TDD cycle** - run tests, implement, verify pass.

**Step 4: Wire up cart commands in main.go**

- `cart` (no subcommand) calls `GetOrderForm` and displays items
- `cart add` calls `AddToCart`
- `cart remove` calls `UpdateItemQuantity` with qty=0
- `cart clear` calls `RemoveAllItems`

**Step 5: Commit**

```bash
git commit -m "feat: cart add/remove/clear via GraphQL mutation + REST orderForm"
```

---

### Task 8: Delivery Windows

**Files:**
- Create: `internal/vtex/delivery.go`
- Create: `internal/vtex/delivery_test.go`

**Step 1: Write the test**

Test that `GetDeliveryWindows` parses the `shippingData.logisticsInfo[0].slas[0].availableDeliveryWindows` array from the orderForm into a clean `[]DeliveryWindow` with start/end times and price.

**Step 2: Write implementation**

`DeliveryWindow` struct: Index, Start (time.Time), End (time.Time), Price (centavos), PriceDisplay (string like "R$ 7,00" or "Grátis").

`GetDeliveryWindows` calls `GetOrderForm` and extracts the windows. No new API call needed.

**Step 3: Wire up** `delivery windows` command. Human output: formatted table. JSON: array of window objects.

**Step 4: Commit**

```bash
git commit -m "feat: delivery windows listing from orderForm SLA data"
```

---

### Task 9: Checkout (Set Shipping + Payment + Place Order)

**Files:**
- Create: `internal/vtex/checkout.go`
- Create: `internal/vtex/checkout_test.go`

**Step 1: Write the test**

Test `SetShippingWindow`, `SetPayment`, and `PlaceOrder` against mock servers that verify the correct REST endpoints and payloads.

**Step 2: Write implementation**

- `SetShippingWindow(orderFormID string, window DeliveryWindow) error` - `POST /api/checkout/pub/orderForm/{id}/attachments/shippingData` with selected SLA + delivery window
- `SetPayment(orderFormID string, paymentSystemID int, value int) error` - `POST /api/checkout/pub/orderForm/{id}/attachments/paymentData`
- `PlaceOrder(orderFormID string) (string, error)` - `POST /api/checkout/pub/orderForm/{id}/transaction` → returns order ID

**Step 3: Wire up checkout command**

The `checkout` command flow:
1. Get orderForm (verify items exist, check min order R$100)
2. If `--window` provided, set shipping window
3. If `--payment` provided, set payment (default: Pix 125)
4. Display order summary
5. If `--confirm` flag present, call `PlaceOrder`
6. Without `--confirm`, print summary + "use --confirm to place order"

Exit code 6 if total < R$100 (AGENDADA minimum).

**Step 4: Commit**

```bash
git commit -m "feat: checkout with delivery window selection and order placement"
```

---

### Task 10: AGENTS.md + Polish

**Files:**
- Create: `AGENTS.md`
- Modify: `cmd/zonasul/main.go` (add `agent exit-codes` subcommand)

**Step 1: Write AGENTS.md**

Document all commands, flags, exit codes, JSON output schemas, and example workflows for AI agent self-discovery. Follow gogcli AGENTS.md format.

**Step 2: Add `agent exit-codes` command**

Prints exit code table in plain text (for agent introspection).

**Step 3: Build final binary and smoke test**

```bash
go build -o zonasul ./cmd/zonasul
./zonasul --help
./zonasul agent exit-codes
./zonasul auth status
./zonasul search banana --json --limit 5
```

**Step 4: Commit**

```bash
git commit -m "feat: AGENTS.md and agent self-discovery command"
```

---

### Task 11: Integration Test (Live API, Optional)

**Files:**
- Create: `internal/vtex/integration_test.go`

Build-tagged `//go:build integration` tests that hit the real Zona Sul API:
1. Search for "banana" and verify results
2. Auth check (requires stored token)
3. Get orderForm

Run with: `go test ./internal/vtex/ -tags integration -v`

Not run in CI - manual verification only.

**Commit:**

```bash
git commit -m "test: live API integration tests (build tag: integration)"
```
