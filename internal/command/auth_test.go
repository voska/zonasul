package command

import (
	"encoding/json"
	"errors"
	"github.com/alecthomas/kong"
	"github.com/voska/vtexkit/cli/errfmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/voska/vtexkit/cli"
	"github.com/voska/vtexkit/store"
	"github.com/voska/zonasul/internal/bff"
	"github.com/zalando/go-keyring"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}

func globals() *cli.Globals {
	return &cli.Globals{CLI: &cli.CLI{}, Store: store.Store{Name: "zonasul"}}
}

func TestLoginSavesOnlyVerifiedBFFSession(t *testing.T) {
	keyring.MockInit()
	if err := keyring.Set("zonasul-cli", "vtex-jwt", "legacy-token"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			_, _ = w.Write([]byte(`{"success":true,"data":{"token":"synthetic-token"}}`))
		case "/users/me":
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"synthetic-user","email":"test@example.invalid"}}`))
		default:
			t.Error("legacy auth must not be called")
		}
	}))
	defer srv.Close()
	r := Runner{Client: bff.New(srv.URL, srv.Client())}
	g := globals()
	g.CLI.Auth.Login.Email = "test@example.invalid"
	g.CLI.Auth.Login.Password = "synthetic-password"
	handled, err := r.Run("auth login", g)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	raw, err := keyring.Get("zonasul-cli", "bff-session")
	if err != nil {
		t.Fatal(err)
	}
	var session Session
	if json.Unmarshal([]byte(raw), &session) != nil || session.Token != "synthetic-token" || session.Email != "test@example.invalid" || session.Password != "synthetic-password" {
		t.Fatal("BFF session was not stored atomically")
	}
	old, err := keyring.Get("zonasul-cli", "vtex-jwt")
	if err != nil || old != "legacy-token" {
		t.Fatal("BFF overwrote legacy VTEX credentials")
	}
}

func TestUnverifiedTokenNeverReplacesSavedSession(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", "existing-session")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"data":{"id":"synthetic-user","email":"other@example.invalid"}}`))
	}))
	defer srv.Close()
	r := Runner{Client: bff.New(srv.URL, srv.Client())}
	g := globals()
	g.CLI.Auth.Login.Email = "test@example.invalid"
	g.CLI.Auth.Login.Token = "unverified-token"
	_, err := r.Run("auth login", g)
	raw, _ := keyring.Get("zonasul-cli", "bff-session")
	if err == nil || raw != "existing-session" {
		t.Fatal("unverified token changed stored session")
	}
}

func TestPendingEmailCannotSwitchAccounts(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-pending", `{"email":"test@example.invalid","userId":"synthetic-user"}`)
	r := Runner{Client: bff.New("http://127.0.0.1:1", nil)}
	g := globals()
	g.CLI.Auth.Code.Verify.Email = "other@example.invalid"
	g.CLI.Auth.Code.Verify.Code = "123456"
	_, err := r.Run("auth code verify", g)
	if err == nil {
		t.Fatal("accepted a different email than the pending login")
	}
}

func TestLegacyCheckoutCannotRunWithBFFSession(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"synthetic-token"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("disabled command made a request: %s", r.URL.Path)
	}))
	defer srv.Close()
	for _, cmd := range []string{"cart reorder <order-id>", "delivery simulate", "list order <name>", "fav order", "subs show"} {
		g := globals()
		handled, err := (&Runner{Client: bff.New(srv.URL, srv.Client())}).Run(cmd, g)
		if !handled || err == nil {
			t.Errorf("legacy command %q fell through", cmd)
		}
	}
	for _, cmd := range []string{"schema <command>", "list show <name>"} {
		handled, err := (&Runner{}).Run(cmd, globals())
		if handled || err != nil {
			t.Errorf("local command %q stopped working", cmd)
		}
	}
}

func TestExpiredBFFSessionRefreshesOnlyThroughBFF(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"expired-token","password":"synthetic-password"}`)
	logins := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me":
			if r.Header.Get("Authorization") == "Bearer expired-token" {
				w.WriteHeader(401)
				return
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"synthetic-user","email":"test@example.invalid"}}`))
		case "/login":
			logins++
			_, _ = w.Write([]byte(`{"success":true,"data":{"token":"refreshed-token"}}`))
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	session, err := (&Runner{Client: bff.New(srv.URL, srv.Client())}).requireAuth(globals())
	if err != nil || session.Token != "refreshed-token" || logins != 1 {
		t.Fatalf("logins=%d err=%v", logins, err)
	}
	raw, _ := keyring.Get("zonasul-cli", "bff-session")
	var stored Session
	if json.Unmarshal([]byte(raw), &stored) != nil || stored.Token != "refreshed-token" {
		t.Fatal("refreshed token was not persisted")
	}
}

func TestEmailLoginReplacesPasswordSessionAndClearsPending(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"old-token","password":"old-password"}`)
	_ = keyring.Set("zonasul-cli", "bff-pending", `{"email":"test@example.invalid","userId":"synthetic-user"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login/email":
			_, _ = w.Write([]byte(`{"success":true,"data":{"token":"email-token"}}`))
		case "/users/me":
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"synthetic-user","email":"test@example.invalid"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	g := globals()
	g.CLI.Auth.Code.Verify.Code = "123456"
	_, err := (&Runner{Client: bff.New(srv.URL, srv.Client())}).Run("auth code verify", g)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := keyring.Get("zonasul-cli", "bff-session")
	var session Session
	if json.Unmarshal([]byte(raw), &session) != nil || session.Token != "email-token" || session.Password != "" {
		t.Fatal("email-code session retained a password")
	}
	if _, err := keyring.Get("zonasul-cli", "bff-pending"); err != keyring.ErrNotFound {
		t.Fatal("pending login was not cleared")
	}
}

func TestTransientIdentityFailureDoesNotReplayPassword(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"synthetic-token","password":"synthetic-password"}`)
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(http.StatusServiceUnavailable) }))
	defer srv.Close()
	_, err := (&Runner{Client: bff.New(srv.URL, srv.Client())}).requireAuth(globals())
	if err == nil || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}

func TestKongCommandRouting(t *testing.T) {
	var args cli.CLI
	parser, err := kong.New(&args, Options("test")...)
	if err != nil {
		t.Fatal(err)
	}
	legacy := map[string]bool{"list show": true, "list add": true, "list remove": true, "list delete": true, "fav show": true, "fav add": true, "fav remove": true, "schema": true, "store info": true}
	handled := map[string]bool{"auth login": true, "auth status": true, "auth code send": true, "auth code verify": true, "auth logout": true, "doctor": true, "search": true, "product": true, "cart show": true, "cart add": true, "cart update": true, "cart remove": true, "cart clear": true, "delivery windows": true, "checkout payments": true, "checkout run": true, "orders": true, "exit-codes": true, "agent exit-codes": true}
	disabled := map[string]bool{"cart reorder": true, "list order": true, "fav order": true, "delivery simulate": true, "subs show": true, "subs pause": true, "subs resume": true, "subs skip": true, "subs unskip": true}
	seen := 0
	var walk func(*kong.Node, string)
	walk = func(node *kong.Node, path string) {
		children := 0
		for _, child := range node.Children {
			if child.Type == kong.CommandNode {
				children++
				walk(child, strings.TrimSpace(path+" "+child.Name))
			}
		}
		if children > 0 || node == parser.Model.Node {
			return
		}
		if !legacy[path] && !handled[path] && !disabled[path] {
			t.Errorf("command has no explicit migration policy: %s", path)
		}
		if legacyAllowed(path) != legacy[path] {
			t.Errorf("wrong fallback for %s", path)
		}
		seen++
	}
	walk(parser.Model.Node, "")
	if seen != len(legacy)+len(handled)+len(disabled) {
		t.Fatalf("tree/policy mismatch: %d leaves", seen)
	}
	if legacyAllowed("new-command") || legacyAllowed("cart new-action") {
		t.Fatal("unknown command fell through to VTEX")
	}
}

func TestLegacyTokenEnvironmentDoesNotOverridePassword(t *testing.T) {
	t.Setenv("ZONASUL_TOKEN", "legacy-vtex-token")
	t.Setenv("ZONASUL_BFF_TOKEN", "")
	t.Setenv("ZONASUL_EMAIL", "test@example.invalid")
	t.Setenv("ZONASUL_PASSWORD", "synthetic-password")
	var args cli.CLI
	parser, err := kong.New(&args, Options("test")...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parser.Parse([]string{"auth", "login", "--no-input"}); err != nil {
		t.Fatal(err)
	}
	if args.Auth.Login.Token != "" || args.Auth.Login.Email != "test@example.invalid" || args.Auth.Login.Password != "synthetic-password" {
		t.Fatal("wrong auth environment bindings")
	}
}

func output(t *testing.T, run func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	defer func() { os.Stdout = old; _ = reader.Close(); _ = writer.Close() }()
	os.Stdout = writer
	run()
	_ = writer.Close()
	os.Stdout = old
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestStatusReportsRefreshableExpiredSession(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"expired-token","password":"synthetic-password"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer srv.Close()
	g := globals()
	g.CLI.JSON = true
	text := output(t, func() {
		if err := (&Runner{Client: bff.New(srv.URL, srv.Client())}).status(g); err != nil {
			t.Fatal(err)
		}
	})
	var status struct {
		Status      string
		Refreshable bool
		Backend     string
	}
	if json.Unmarshal([]byte(text), &status) != nil || status.Status != "expired" || !status.Refreshable || status.Backend != "bff" {
		t.Fatalf("status=%s", text)
	}
	if strings.Contains(text, "expired-token") || strings.Contains(text, "synthetic-password") {
		t.Fatal("secret in status output")
	}
}

func TestDoctorDoesNotPrescribeAddressRestoreOnReadFailure(t *testing.T) {
	keyring.MockInit()
	_ = keyring.Set("zonasul-cli", "bff-session", `{"email":"test@example.invalid","token":"synthetic-token"}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/me":
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"synthetic-user","email":"test@example.invalid"}}`))
		case "/addresses":
			w.WriteHeader(503)
		case "/payment-methods":
			_, _ = w.Write([]byte(`{"success":true,"data":[]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	g := globals()
	g.CLI.JSON = true
	var runErr error
	text := output(t, func() { runErr = (&Runner{Client: bff.New(srv.URL, srv.Client())}).doctor(g) })
	var result struct {
		Ready  bool
		Checks []check
	}
	if json.Unmarshal([]byte(text), &result) != nil || result.Ready || runErr == nil {
		t.Fatalf("report=%s err=%v", text, runErr)
	}
	for _, c := range result.Checks {
		if c.Name == "delivery address" && (c.OK || c.Fix != "") {
			t.Fatal("failed read was reported as a missing address")
		}
	}
}

func TestOversizedKeyringEntryReportsConfigError(t *testing.T) {
	keyring.MockInitWithError(keyring.ErrSetDataTooBig)
	t.Cleanup(keyring.MockInit)
	err := saveSecret(globals(), "bff-session", Session{Token: "synthetic-token"})
	var typed *errfmt.Error
	if !errors.As(err, &typed) || typed.Code != 10 || !strings.Contains(typed.Message, "size limit") {
		t.Fatalf("err=%v", err)
	}
}

func TestDisabledCommandsUseDomainExitCode(t *testing.T) {
	_, err := (&Runner{}).Run("subs show", globals())
	var typed *errfmt.Error
	if !errors.As(err, &typed) || typed.Code != 9 || !strings.Contains(typed.Message, "nothing was sent") {
		t.Fatalf("err=%v", err)
	}
}

func TestBFFTokenEnvironmentIsBound(t *testing.T) {
	t.Setenv("ZONASUL_BFF_TOKEN", "synthetic-bff-token")
	var args cli.CLI
	parser, err := kong.New(&args, Options("test")...)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = parser.Parse([]string{"auth", "login", "--email", "test@example.invalid"}); err != nil {
		t.Fatal(err)
	}
	if args.Auth.Login.Token != "synthetic-bff-token" {
		t.Fatal("BFF token environment was not bound")
	}
}
