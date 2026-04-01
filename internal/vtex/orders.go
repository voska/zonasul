package vtex

import (
	"encoding/json"
	"fmt"
)

type Order struct {
	OrderID           string  `json:"orderId"`
	CreationDate      string  `json:"creationDate"`
	Status            string  `json:"status"`
	StatusDescription string  `json:"statusDescription"`
	TotalValue        float64 `json:"totalValue"`
	TotalItems        int     `json:"totalItems"`
}

func (c *Client) ListOrders() ([]Order, error) {
	body, err := c.Get("/api/oms/user/orders")
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	var resp struct {
		List []Order `json:"list"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("list orders parse: %w", err)
	}
	return resp.List, nil
}

type OrderItem struct {
	ID       string `json:"id"`
	SKU      string `json:"sellerSku"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type OrderDetail struct {
	OrderID string      `json:"orderId"`
	Items   []OrderItem `json:"items"`
}

func (c *Client) GetOrder(orderID string) (*OrderDetail, error) {
	body, err := c.Get(fmt.Sprintf("/api/oms/user/orders/%s", orderID))
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	var detail OrderDetail
	if err := json.Unmarshal(body, &detail); err != nil {
		return nil, fmt.Errorf("get order parse: %w", err)
	}
	return &detail, nil
}
