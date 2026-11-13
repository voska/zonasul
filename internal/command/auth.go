// Package command routes the shared CLI surface to Zona Sul's new storefront.
package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/voska/vtexkit/cli"
	"github.com/voska/vtexkit/cli/config"
	"github.com/voska/vtexkit/cli/errfmt"
	"github.com/voska/vtexkit/cli/outfmt"
	"github.com/voska/zonasul/internal/bff"
	"github.com/zalando/go-keyring"
)

// Session is stored as one OS-keyring entry, separate from all VTEX secrets.
type Session struct {
	Email    string `json:"email"`
	Token    string `json:"token"`
	Password string `json:"password,omitempty"`
}

type pending struct {
	Email  string `json:"email"`
	UserID string `json:"userId"`
}

type Runner struct{ Client *bff.Client }

func (r *Runner) client() *bff.Client {
	if r.Client == nil {
		r.Client = bff.New(bff.BaseURL, nil)
	}
	return r.Client
}

// Run returns false only for commands that can still use the shared library.
func (r *Runner) Run(command string, g *cli.Globals) (bool, error) {
	name := strings.Split(command, " <")[0]
	switch name {
	case "exit-codes", "agent exit-codes":
		codes := append(errfmt.ExitCodeTable(), errfmt.ExitCodeEntry{Code: bff.ExitSubmittedUnverified, Name: "submitted_unverified", Description: "Payment attempted; do not retry; verify issuer settlement", Transient: false})
		return true, g.Formatter().Print(codes)
	case "search":
		return true, r.catalog(g, false)
	case "product":
		return true, r.catalog(g, true)
	case "cart show", "cart add", "cart update", "cart remove", "cart clear":
		return true, r.cart(g, strings.TrimPrefix(name, "cart "))
	case "delivery windows":
		return true, r.delivery(g)
	case "checkout payments":
		return true, r.payments(g)
	case "checkout run":
		return true, r.checkout(g)
	case "orders":
		return true, r.orders(g)
	case "doctor":
		return true, r.doctor(g)
	case "auth login":
		return true, r.login(g)
	case "auth status":
		return true, r.status(g)
	case "auth code send":
		return true, r.send(g)
	case "auth code verify":
		return true, r.verify(g)
	case "auth logout":
		return true, r.logout(g)
	}

	if legacyAllowed(command) {
		return false, nil
	}
	return true, errfmt.Domain("command is disabled during the storefront migration; nothing was sent and no cart or order changed")
}

func legacyAllowed(command string) bool {
	name := strings.Split(command, " <")[0]
	switch name {
	case "list show", "list add", "list remove", "list delete", "fav show", "fav add", "fav remove", "store info", "schema":
		return true
	}
	return false
}

func readSecret(g *cli.Globals, key string, value any) error {
	raw, err := keyring.Get(g.Store.KeyringService(), key)
	if errors.Is(err, keyring.ErrNotFound) {
		return errfmt.Auth("no BFF session; run zonasul auth login")
	}
	if err != nil {
		return errfmt.Config("could not read Zona Sul keyring")
	}
	if json.Unmarshal([]byte(raw), value) != nil {
		return errfmt.Config("invalid Zona Sul keyring record")
	}
	return nil
}
func saveSecret(g *cli.Globals, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return errfmt.Config("could not encode Zona Sul session")
	}
	if err := keyring.Set(g.Store.KeyringService(), key, string(data)); err != nil {
		if errors.Is(err, keyring.ErrSetDataTooBig) {
			return errfmt.Config("BFF session exceeds the OS-keyring size limit")
		}
		return errfmt.Config("could not store Zona Sul session in OS keyring")
	}
	return nil
}
func deleteSecret(g *cli.Globals, key string) error {
	err := keyring.Delete(g.Store.KeyringService(), key)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return errfmt.Config("could not clear Zona Sul keyring record")
	}
	return nil
}
func authError(err error) bool {
	var e *errfmt.Error
	return errors.As(err, &e) && e.Code == errfmt.ExitAuth
}

func (r *Runner) login(g *cli.Globals) error {
	c := g.CLI.Auth.Login
	email, password := c.Email, c.Password
	if email == "" {
		var err error
		email, err = g.Prompt("Email: ")
		if err != nil {
			return err
		}
	}
	var token string
	if c.Token != "" {
		if _, err := r.client().Identity(c.Token, email); err != nil {
			return err
		}
		token = c.Token
		password = ""
	} else {
		if password == "" {
			var err error
			password, err = g.Prompt("Password: ")
			if err != nil {
				return err
			}
		}
		var err error
		token, err = r.client().Login(email, password)
		if err != nil {
			return err
		}
	}
	if err := saveSecret(g, "bff-session", Session{Email: email, Token: token, Password: password}); err != nil {
		return err
	}
	outfmt.Success("Logged in as %s. Run doctor to check checkout readiness.", email)
	return nil
}

func (r *Runner) send(g *cli.Globals) error {
	email := g.CLI.Auth.Code.Send.Email
	id, err := r.client().SendCode(email)
	if err != nil {
		return err
	}
	if err := saveSecret(g, "bff-pending", pending{Email: email, UserID: id}); err != nil {
		return err
	}
	outfmt.Success("Zona Sul accepted the login-code request; delivery is unverified. Run: zonasul auth code verify --code <code>")
	return nil
}
func (r *Runner) verify(g *cli.Globals) error {
	var p pending
	if err := readSecret(g, "bff-pending", &p); err != nil {
		if authError(err) {
			return errfmt.Auth("no pending BFF email login; run zonasul auth code send")
		}
		return err
	}
	c := g.CLI.Auth.Code.Verify
	if p.Email == "" || p.UserID == "" {
		return errfmt.Auth("no pending BFF email login; run zonasul auth code send")
	}
	if c.Email != "" && !strings.EqualFold(c.Email, p.Email) {
		return errfmt.Usage("email differs from the pending BFF login")
	}
	token, err := r.client().VerifyCode(p.Email, p.UserID, c.Code)
	if err != nil {
		return err
	}
	if err := saveSecret(g, "bff-session", Session{Email: p.Email, Token: token}); err != nil {
		return err
	}
	if err := deleteSecret(g, "bff-pending"); err != nil {
		return err
	}
	outfmt.Success("Logged in to the Zona Sul storefront. Email-code sessions require another code when they expire.")
	return nil
}

func (r *Runner) status(g *cli.Globals) error {
	var s Session
	state := "unauthenticated"
	err := readSecret(g, "bff-session", &s)
	if err != nil && !authError(err) {
		return err
	}
	if err == nil {
		if _, err := r.client().Identity(s.Token, s.Email); err == nil {
			state = "authenticated"
		} else if authError(err) {
			state = "expired"
		} else {
			return err
		}
	}
	result := map[string]any{"store": "zonasul", "email": s.Email, "status": state, "refreshable": s.Password != "", "backend": "bff"}
	if state == "authenticated" {
		result["user"] = s.Email
	}
	if g.CLI.JSON || g.CLI.Plain || g.CLI.Quiet {
		return g.Formatter().Print(result)
	}
	fmt.Printf("%-12s %s\n", "store", g.Store.Label())
	fmt.Printf("%-12s %s\n", "backend", "bff")
	fmt.Printf("%-12s %s\n", "status", state)
	if s.Email != "" {
		fmt.Printf("%-12s %s\n", "email", s.Email)
	}
	if state != "authenticated" {
		outfmt.Hint("Run: zonasul auth login")
	}
	return nil
}

func (r *Runner) requireAuth(g *cli.Globals) (*Session, error) {
	var s Session
	if err := readSecret(g, "bff-session", &s); err != nil {
		return nil, err
	}
	if _, err := r.client().Identity(s.Token, s.Email); err != nil {
		if !authError(err) || s.Password == "" {
			return nil, err
		}
		token, err := r.client().Login(s.Email, s.Password)
		if err != nil {
			return nil, err
		}
		s.Token = token
		if err := saveSecret(g, "bff-session", s); err != nil {
			return nil, err
		}
	}
	return &s, nil
}

func (r *Runner) logout(g *cli.Globals) error {
	var failures []error
	for _, key := range []string{"bff-session", "bff-pending", "vtex-jwt", "login-password"} {
		if err := deleteSecret(g, key); err != nil {
			failures = append(failures, err)
		}
	}
	if err := g.Config().SaveCredentials(&config.Credentials{}); err != nil {
		failures = append(failures, errfmt.Config("could not clear legacy email"))
	}
	if len(failures) > 0 {
		return errors.Join(failures...)
	}
	outfmt.Success("Logged out of the BFF and legacy VTEX sessions.")
	return nil
}
