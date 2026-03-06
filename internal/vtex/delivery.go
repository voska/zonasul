package vtex

import (
	"encoding/json"
	"fmt"
	"time"
)

type DeliveryWindow struct {
	Index    int       `json:"index"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Price    int       `json:"price"`
	LisPrice int       `json:"lisPrice"`
	Tax      int       `json:"tax"`
	RawStart string    `json:"-"`
	RawEnd   string    `json:"-"`
}

type orderFormWithShipping struct {
	OrderFormID  string `json:"orderFormId"`
	ShippingData struct {
		LogisticsInfo []struct {
			SLAs []struct {
				ID                       string `json:"id"`
				AvailableDeliveryWindows []struct {
					StartDateUtc string `json:"startDateUtc"`
					EndDateUtc   string `json:"endDateUtc"`
					Price        int    `json:"price"`
					LisPrice     int    `json:"lisPrice"`
					Tax          int    `json:"tax"`
				} `json:"availableDeliveryWindows"`
			} `json:"slas"`
		} `json:"logisticsInfo"`
	} `json:"shippingData"`
}

func (c *Client) GetDeliveryWindows(orderFormID string) ([]DeliveryWindow, error) {
	path := "/api/checkout/pub/orderForm"
	if orderFormID != "" {
		path = fmt.Sprintf("/api/checkout/pub/orderForm/%s", orderFormID)
	}
	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get delivery windows: %w", err)
	}

	var of orderFormWithShipping
	if err := json.Unmarshal(body, &of); err != nil {
		return nil, fmt.Errorf("delivery windows parse: %w", err)
	}

	seen := map[string]bool{}
	var windows []DeliveryWindow
	idx := 0
	for _, li := range of.ShippingData.LogisticsInfo {
		for _, sla := range li.SLAs {
			for _, dw := range sla.AvailableDeliveryWindows {
				key := dw.StartDateUtc + "|" + dw.EndDateUtc
				if seen[key] {
					continue
				}
				seen[key] = true
				start, _ := time.Parse(time.RFC3339, dw.StartDateUtc)
				end, _ := time.Parse(time.RFC3339, dw.EndDateUtc)
				windows = append(windows, DeliveryWindow{
					Index:    idx,
					Start:    start,
					End:      end,
					Price:    dw.Price,
					LisPrice: dw.LisPrice,
					Tax:      dw.Tax,
					RawStart: dw.StartDateUtc,
					RawEnd:   dw.EndDateUtc,
				})
				idx++
			}
		}
	}
	return windows, nil
}
