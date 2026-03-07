package vtex

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"runtime"
)

const (
	clearSaleAppKey = "b5qhnn79ksdoeru452lt"
	clearSaleBase   = "https://device.clearsale.com.br/p"
	clearSaleSDK    = "@clear.sale/behavior-analytics-fingerprint-sdk"
	clearSaleVer    = "1.0.253"
)

// GenerateClearSaleSession creates a ClearSale anti-fraud session by
// generating a fingerprint session ID and registering device data with
// ClearSale's collection endpoints (fp1.png, fp2.png).
// Returns the session ID to use as deviceFingerprint in payment requests.
func GenerateClearSaleSession() (string, error) {
	sid := generateSessionID()

	// Register fingerprint data with ClearSale
	if err := sendFingerprint1(sid); err != nil {
		return sid, nil // non-fatal: session ID is still usable
	}
	if err := sendFingerprint2(sid); err != nil {
		return sid, nil
	}

	return sid, nil
}

func generateSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:16])
}

// sendFingerprint1 sends canvas/WebGL hashes to ClearSale's fp1.png endpoint.
func sendFingerprint1(sid string) error {
	// Generate plausible hashes (ClearSale uses these for device identity)
	bb := hashString("canvas-cli-" + sid)
	ba := hashString("webgl-cli-" + sid)
	a2 := hashString("fonts-cli-" + sid)

	params := url.Values{
		"bb":  {bb},
		"ba":  {ba},
		"a2":  {a2},
		"app": {clearSaleAppKey},
		"sid": {sid},
		"id":  {clearSaleSDK},
		"v":   {clearSaleVer},
		"sm":  {"true"},
	}

	resp, err := http.Get(clearSaleBase + "/fp1.png?" + params.Encode())
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}

// sendFingerprint2 sends device telemetry to ClearSale's fp2.png endpoint.
func sendFingerprint2(sid string) error {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36"
	platform := "MacIntel"
	if runtime.GOARCH == "arm64" {
		platform = "MacIntel" // Chrome still reports MacIntel on Apple Silicon
	}

	params := url.Values{
		"aa":  {ua},
		"ab":  {"pt-BR"},
		"ac":  {"30"},      // color depth
		"ad":  {"2"},       // pixel ratio
		"ae":  {"1080"},    // screen height
		"af":  {"1920"},    // screen width
		"ag":  {"1048"},    // available height
		"ah":  {"1920"},    // available width
		"ai":  {"180"},     // timezone offset (BRT = UTC-3 = 180 min)
		"aj":  {"1"},       // cookieEnabled
		"ak":  {"1"},       // localStorage
		"al":  {"1"},       // sessionStorage
		"am":  {"0"},       // indexedDB
		"an":  {"0"},       // openDatabase
		"ao":  {"unknown"}, // plugins
		"ap":  {platform},  // platform
		"aq":  {"unknown"}, // doNotTrack
		"ar":  {hashString("canvas-render-" + sid)},
		"as":  {hashString("webgl-render-" + sid)},
		"at":  {"0"},
		"ay":  {hashString("fonts-detect-" + sid)},
		"a3":  {"10"},
		"m1":  {"1"},
		"mb":  {"0"},
		"hd":  {"0"},
		"mr":  {"8"},
		"h1":  {hashString("audio-" + sid)},
		"h6":  {hashString("webgl-ext-" + sid)},
		"h4":  {hashString("webgl-params-" + sid)},
		"l1":  {"0"},
		"im":  {"0"},
		"b2":  {"0.82"},
		"b1":  {"0"},
		"az":  {hashString("misc-" + sid)},
		"h7":  {hashString("webgl-vendor-" + sid)},
		"app": {clearSaleAppKey},
		"sid": {sid},
		"id":  {clearSaleSDK},
		"v":   {clearSaleVer},
	}

	resp, err := http.Get(clearSaleBase + "/fp2.png?" + params.Encode())
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	return nil
}
