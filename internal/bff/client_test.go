package bff

import (
	"encoding/json"
	"errors"
	"github.com/voska/vtexkit/cli/errfmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginRequiresMatchingLiveIdentity(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		ok          bool
	}{
		{"matching", `{"id":"synthetic-user","email":"test@example.invalid"}`, true},
		{"wrong identity", `{"id":"synthetic-user","email":"other@example.invalid"}`, false},
		{"missing identity", `{"email":"test@example.invalid"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("x-contractaccountid") != Contract {
					t.Error("missing contract")
				}
				switch r.URL.Path {
				case "/login":
					if r.Method != http.MethodPost {
						t.Error("wrong login method")
					}
					var payload map[string]string
					_ = json.NewDecoder(r.Body).Decode(&payload)
					if payload["login"] != "test@example.invalid" || payload["password"] != "synthetic-password" {
						t.Error("wrong login payload")
					}
					_, _ = w.Write([]byte(`{"success":true,"data":{"token":"synthetic-token"}}`))
				case "/users/me":
					if r.Header.Get("Authorization") != "Bearer synthetic-token" {
						t.Error("missing bearer")
					}
					if len(r.Cookies()) != 0 {
						t.Error("must not send VTEX cookies")
					}
					_, _ = w.Write([]byte(`{"success":true,"data":` + tc.reply + `}`))
				default:
					t.Errorf("unexpected endpoint %s", r.URL.Path)
				}
			}))
			defer srv.Close()
			c := New(srv.URL, srv.Client())
			token, err := c.Login("test@example.invalid", "synthetic-password")
			if (err == nil) != tc.ok {
				t.Fatalf("err=%v", err)
			}
			if tc.ok && token != "synthetic-token" {
				t.Error("token lost")
			}
			if !tc.ok && token != "" {
				t.Error("returned unverified token")
			}
		})
	}
}

func TestRequestRejectsFailureWithoutLeakingValues(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
	}{
		{200, `{"success":false,"message":"private-value"}`},
		{200, `{"success":true,"data":null}`},
		{200, `<html>private-value</html>`},
		{401, `{"message":"private-value"}`},
		{429, `{"message":"private-value"}`},
		{503, `private-value`},
	} {
		t.Run(tc.body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			var result map[string]any
			err := New(srv.URL, srv.Client()).Request(http.MethodGet, "/users/me", "synthetic-token", nil, &result)
			if err == nil || strings.Contains(err.Error(), "private-value") {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestEmailLoginUsesLoginCodeRoute(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/login/email/send-code":
			_, _ = w.Write([]byte(`{"success":true,"data":{"userId":"synthetic-user"}}`))
		case "/login/email":
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)
			if payload["userId"] != "synthetic-user" || payload["code"] != "123456" {
				t.Error("pending userId/code lost")
			}
			_, _ = w.Write([]byte(`{"success":true,"data":{"token":"synthetic-token"}}`))
		case "/users/me":
			_, _ = w.Write([]byte(`{"success":true,"data":{"id":"synthetic-user","email":"test@example.invalid"}}`))
		default:
			t.Error("unexpected endpoint")
		}
	}))
	defer srv.Close()
	c := New(srv.URL, srv.Client())
	id, err := c.SendCode("test@example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.VerifyCode("test@example.invalid", id, "123456")
	if err != nil || token != "synthetic-token" {
		t.Fatalf("token empty=%v err=%v", token == "", err)
	}
	if len(paths) != 3 {
		t.Fatalf("paths=%v", paths)
	}
}

func TestAuthenticatedRedirectIsNotFollowed(t *testing.T) {
	followed := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { followed = true }))
	defer target.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer origin.Close()
	_, err := New(origin.URL, origin.Client()).Identity("synthetic-token", "test@example.invalid")
	if err == nil || followed {
		t.Fatal("followed authenticated redirect")
	}
}

func TestLoginRejectsCredentialsWithAuthExitCode(t *testing.T) {
	for _, status := range []int{400, 401, 422} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"message":"private-value"}`))
		}))
		_, err := New(srv.URL, srv.Client()).Login("test@example.invalid", "synthetic-password")
		srv.Close()
		var typed *errfmt.Error
		if !errors.As(err, &typed) || typed.Code != 4 || strings.Contains(typed.Message, "private-value") {
			t.Fatalf("HTTP %d: %v", status, err)
		}
	}
}
