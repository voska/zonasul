package vtex_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/voska/zonasul/internal/vtex"
)

func TestGetDeliveryWindows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"orderFormId": "abc123",
			"shippingData": map[string]any{
				"logisticsInfo": []map[string]any{
					{
						"slas": []map[string]any{
							{
								"id": "AGENDADA",
								"availableDeliveryWindows": []map[string]any{
									{
										"startDateUtc": "2026-03-04T14:01:00+00:00",
										"endDateUtc":   "2026-03-04T16:00:59+00:00",
										"price":        700,
										"lisPrice":     0,
										"tax":          0,
									},
									{
										"startDateUtc": "2026-03-04T13:00:00+00:00",
										"endDateUtc":   "2026-03-04T18:00:59+00:00",
										"price":        0,
										"lisPrice":     0,
										"tax":          0,
									},
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

	c := vtex.NewClient(srv.URL, "test-jwt")
	windows, err := c.GetDeliveryWindows("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if len(windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(windows))
	}
	if windows[0].Price != 700 {
		t.Errorf("expected price 700, got %d", windows[0].Price)
	}
	if windows[1].Price != 0 {
		t.Errorf("expected price 0 (free), got %d", windows[1].Price)
	}
	if windows[0].Start.Hour() != 14 {
		t.Errorf("expected start hour 14, got %d", windows[0].Start.Hour())
	}
	if windows[0].LisPrice != 0 {
		t.Errorf("expected lisPrice 0, got %d", windows[0].LisPrice)
	}
	if windows[0].Tax != 0 {
		t.Errorf("expected tax 0, got %d", windows[0].Tax)
	}
}
