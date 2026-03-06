package vtex

import (
	"encoding/json"
	"fmt"
)

type OrderFormItem struct {
	ID           string `json:"id"`
	ProductID    string `json:"productId"`
	Name         string `json:"name"`
	Quantity     int    `json:"quantity"`
	Price        int    `json:"price"`
	SellingPrice int    `json:"sellingPrice"`
	Seller       string `json:"seller"`
}

type Totalizer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type OrderForm struct {
	OrderFormID string          `json:"orderFormId"`
	Items       []OrderFormItem `json:"items"`
	Totalizers  []Totalizer     `json:"totalizers"`
}

func (c *Client) AddToCart(orderFormID, skuID string, quantity int) (*OrderForm, error) {
	if orderFormID == "" {
		of, err := c.GetOrderForm("")
		if err != nil {
			return nil, fmt.Errorf("add to cart: get orderForm: %w", err)
		}
		orderFormID = of.OrderFormID
	}

	payload := map[string]any{
		"orderItems": []map[string]any{
			{
				"id":       skuID,
				"quantity": quantity,
				"seller":   SellerID,
			},
		},
	}

	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/items", orderFormID)
	body, err := c.PostJSON(path, payload)
	if err != nil {
		return nil, fmt.Errorf("add to cart: %w", err)
	}

	var result OrderForm
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("add to cart parse: %w", err)
	}
	return &result, nil
}

func (c *Client) GetOrderForm(orderFormID string) (*OrderForm, error) {
	path := "/api/checkout/pub/orderForm"
	if orderFormID != "" {
		path = fmt.Sprintf("/api/checkout/pub/orderForm/%s", orderFormID)
	}
	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("get order form: %w", err)
	}
	var of OrderForm
	if err := json.Unmarshal(body, &of); err != nil {
		return nil, fmt.Errorf("get order form parse: %w", err)
	}
	return &of, nil
}

func (c *Client) UpdateItemQuantity(orderFormID string, index, quantity int) (*OrderForm, error) {
	payload := map[string]any{
		"orderItems": []map[string]any{
			{
				"index":    index,
				"quantity": quantity,
			},
		},
	}
	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/items/update", orderFormID)
	body, err := c.PostJSON(path, payload)
	if err != nil {
		return nil, fmt.Errorf("update item: %w", err)
	}
	var of OrderForm
	if err := json.Unmarshal(body, &of); err != nil {
		return nil, fmt.Errorf("update item parse: %w", err)
	}
	return &of, nil
}

func (c *Client) RemoveAllItems(orderFormID string) error {
	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/items/removeAll", orderFormID)
	_, err := c.PostJSON(path, map[string]any{})
	if err != nil {
		return fmt.Errorf("remove all items: %w", err)
	}
	return nil
}
