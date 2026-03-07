package vtex

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type shippingInfo struct {
	Address           map[string]any   `json:"address"`
	SelectedAddresses []map[string]any `json:"selectedAddresses"`
	SLAName           string
	AddressID         string
	NumItems          int
}

func (c *Client) getShippingInfo(orderFormID string) (*shippingInfo, error) {
	body, err := c.Get(fmt.Sprintf("/api/checkout/pub/orderForm/%s", orderFormID))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Items        []any `json:"items"`
		ShippingData struct {
			Address            map[string]any   `json:"address"`
			SelectedAddresses  []map[string]any `json:"selectedAddresses"`
			AvailableAddresses []map[string]any `json:"availableAddresses"`
			LogisticsInfo      []struct {
				SLAs []struct {
					ID string `json:"id"`
				} `json:"slas"`
			} `json:"logisticsInfo"`
		} `json:"shippingData"`
	}
	_ = json.Unmarshal(body, &resp)

	info := &shippingInfo{NumItems: len(resp.Items)}

	if len(resp.ShippingData.SelectedAddresses) > 0 {
		info.SelectedAddresses = resp.ShippingData.SelectedAddresses
		info.Address = resp.ShippingData.SelectedAddresses[0]
	} else if resp.ShippingData.Address != nil {
		info.Address = resp.ShippingData.Address
		info.SelectedAddresses = []map[string]any{resp.ShippingData.Address}
	} else if len(resp.ShippingData.AvailableAddresses) > 0 {
		info.Address = resp.ShippingData.AvailableAddresses[0]
		info.SelectedAddresses = []map[string]any{resp.ShippingData.AvailableAddresses[0]}
	}

	if info.Address != nil {
		if id, ok := info.Address["addressId"].(string); ok {
			info.AddressID = id
		}
	}

	if len(resp.ShippingData.LogisticsInfo) > 0 && len(resp.ShippingData.LogisticsInfo[0].SLAs) > 0 {
		info.SLAName = resp.ShippingData.LogisticsInfo[0].SLAs[0].ID
	}

	return info, nil
}

func (c *Client) SetAddress(orderFormID string, numItems int) error {
	info, err := c.getShippingInfo(orderFormID)
	if err != nil {
		return fmt.Errorf("set address: %w", err)
	}

	if info.Address == nil {
		return fmt.Errorf("set address: no address found on account")
	}

	sla := info.SLAName
	if sla == "" {
		sla = "Entrega Zona Sul"
	}

	logisticsInfo := make([]map[string]any, numItems)
	for i := range numItems {
		logisticsInfo[i] = map[string]any{
			"itemIndex":               i,
			"addressId":               info.AddressID,
			"selectedSla":             sla,
			"selectedDeliveryChannel": "delivery",
		}
	}

	payload := map[string]any{
		"logisticsInfo":                    logisticsInfo,
		"selectedAddresses":                info.SelectedAddresses,
		"clearAddressIfPostalCodeNotFound": false,
	}
	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/attachments/shippingData", orderFormID)
	_, err = c.PostJSON(path, payload)
	if err != nil {
		return fmt.Errorf("set address: %w", err)
	}
	return nil
}

func (c *Client) SetShippingWindow(orderFormID string, window DeliveryWindow, numItems int) error {
	info, err := c.getShippingInfo(orderFormID)
	if err != nil {
		return fmt.Errorf("set shipping: %w", err)
	}

	if info.Address == nil {
		return fmt.Errorf("set shipping: no address found on account")
	}

	sla := info.SLAName
	if sla == "" {
		sla = "Entrega Zona Sul"
	}

	logisticsInfo := make([]map[string]any, numItems)
	for i := range numItems {
		logisticsInfo[i] = map[string]any{
			"itemIndex":               i,
			"addressId":               info.AddressID,
			"selectedSla":             sla,
			"selectedDeliveryChannel": "delivery",
			"deliveryWindow": map[string]any{
				"startDateUtc": window.RawStart,
				"endDateUtc":   window.RawEnd,
				"price":        window.Price,
				"lisPrice":     window.LisPrice,
				"tax":          window.Tax,
			},
		}
	}

	payload := map[string]any{
		"logisticsInfo":                    logisticsInfo,
		"selectedAddresses":                info.SelectedAddresses,
		"clearAddressIfPostalCodeNotFound": false,
	}
	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/attachments/shippingData", orderFormID)
	_, err = c.PostJSON(path, payload)
	if err != nil {
		return fmt.Errorf("set shipping window: %w", err)
	}
	return nil
}

type SavedCard struct {
	AccountID          string `json:"accountId"`
	CardNumber         string `json:"cardNumber"`
	Bin                string `json:"bin"`
	PaymentSystem      string `json:"paymentSystem"`
	PaymentSystemName  string `json:"paymentSystemName"`
	AvailableAddresses []any  `json:"availableAddresses"`
}

func (c *Client) GetSavedCards(orderFormID string) ([]SavedCard, error) {
	body, err := c.Get(fmt.Sprintf("/api/checkout/pub/orderForm/%s", orderFormID))
	if err != nil {
		return nil, fmt.Errorf("get saved cards: %w", err)
	}
	var resp struct {
		PaymentData struct {
			AvailableAccounts []SavedCard `json:"availableAccounts"`
		} `json:"paymentData"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("get saved cards parse: %w", err)
	}
	return resp.PaymentData.AvailableAccounts, nil
}

func (c *Client) SetPayment(orderFormID string, paymentSystemID int, value int) error {
	payload := map[string]any{
		"payments": []map[string]any{
			{
				"paymentSystem":  paymentSystemID,
				"referenceValue": value,
				"value":          value,
				"installments":   1,
			},
		},
	}
	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/attachments/paymentData", orderFormID)
	_, err := c.PostJSON(path, payload)
	if err != nil {
		return fmt.Errorf("set payment: %w", err)
	}
	return nil
}

func (c *Client) SetPaymentWithSavedCard(orderFormID string, card SavedCard, value int) error {
	psID := 2
	if card.PaymentSystem != "" {
		_, _ = fmt.Sscanf(card.PaymentSystem, "%d", &psID)
	}

	payload := map[string]any{
		"payments": []map[string]any{
			{
				"paymentSystem":     psID,
				"paymentSystemName": card.PaymentSystemName,
				"group":             "creditCardPaymentGroup",
				"installments":      1,
				"installmentsValue": value,
				"value":             value,
				"referenceValue":    value,
				"accountId":         card.AccountID,
				"tokenId":           nil,
			},
		},
	}
	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/attachments/paymentData", orderFormID)
	_, err := c.PostJSON(path, payload)
	if err != nil {
		return fmt.Errorf("set payment with saved card: %w", err)
	}
	return nil
}

type TransactionResult struct {
	OrderGroup    string
	TransactionID string
	ReceiverUri   string
	MerchantName  string
}

func (c *Client) PlaceOrder(orderFormID string, orderValue int) (*TransactionResult, error) {
	payload := map[string]any{
		"referenceId":      orderFormID,
		"savePersonalData": true,
		"optinNewsLetter":  false,
		"value":            orderValue,
		"referenceValue":   orderValue,
		"interestValue":    0,
	}
	path := fmt.Sprintf("/api/checkout/pub/orderForm/%s/transaction", orderFormID)
	body, err := c.PostJSON(path, payload)
	if err != nil {
		return nil, fmt.Errorf("place order: %w", err)
	}

	var resp struct {
		ID         string `json:"id"`
		OrderGroup string `json:"orderGroup"`
		Messages   []struct {
			Code   string `json:"code"`
			Text   string `json:"text"`
			Status string `json:"status"`
		} `json:"messages"`
		MerchantTransactions []struct {
			ID            string `json:"id"`
			TransactionID string `json:"transactionId"`
			Payments      []struct {
				ID string `json:"id"`
			} `json:"payments"`
		} `json:"merchantTransactions"`
		ReceiverUri string `json:"receiverUri"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("place order parse: %w", err)
	}

	for _, msg := range resp.Messages {
		if msg.Status == "error" {
			return nil, fmt.Errorf("place order: %s", msg.Text)
		}
	}

	result := &TransactionResult{
		OrderGroup:  resp.OrderGroup,
		ReceiverUri: resp.ReceiverUri,
	}

	if result.OrderGroup == "" {
		result.OrderGroup = resp.ID
	}
	if result.OrderGroup == "" {
		return nil, fmt.Errorf("place order: no order ID in response")
	}

	if len(resp.MerchantTransactions) > 0 {
		mt := resp.MerchantTransactions[0]
		result.TransactionID = mt.TransactionID
		result.MerchantName = mt.ID
	}

	return result, nil
}

func (c *Client) PayWithSavedCard(tx *TransactionResult, card SavedCard, cvv string, orderValue int) error {
	psID := 2
	if card.PaymentSystem != "" {
		_, _ = fmt.Sscanf(card.PaymentSystem, "%d", &psID)
	}

	// tx.MerchantName is the full ID like "ZONASULZSA-zonasulzsa"
	// The payment id field uses this as-is
	// The transaction.merchantName uses just the uppercase part
	paymentID := tx.MerchantName
	if paymentID == "" {
		paymentID = "ZONASULZSA-" + SellerID
	}
	merchantName := strings.Split(paymentID, "-")[0]

	// Generate ClearSale anti-fraud fingerprint session.
	// The payment gateway validates credit card transactions against ClearSale's
	// device fingerprint data. Without this, transactions get Cielo code 59 (suspected fraud).
	csSessionID, _ := GenerateClearSaleSession()

	payload := []map[string]any{
		{
			"paymentSystem":            psID,
			"paymentSystemName":        card.PaymentSystemName,
			"group":                    "creditCardPaymentGroup",
			"installments":             1,
			"installmentsInterestRate": 0,
			"installmentsValue":        orderValue,
			"value":                    orderValue,
			"referenceValue":           orderValue,
			"accountId":                card.AccountID,
			"fields": map[string]string{
				"validationCode":    cvv,
				"securityCode":      cvv,
				"accountId":         card.AccountID,
				"bin":               card.Bin,
				"deviceFingerprint": csSessionID,
			},
			"hasDefaultBillingAddress":  true,
			"isBillingAddressDifferent": false,
			"id":                        paymentID,
			"interestRate":              0,
			"installmentValue":          orderValue,
			"transaction": map[string]string{
				"id":           tx.TransactionID,
				"merchantName": merchantName,
			},
			"currencyCode":         "BRL",
			"originalPaymentIndex": 0,
		},
	}

	callbackURL := fmt.Sprintf("https://www.zonasul.com.br/checkout/gatewayCallback/%s/{messageCode}", tx.OrderGroup)
	paymentsURL := fmt.Sprintf("https://%s.vtexpayments.com.br/api/payments/pub/transactions/%s/payments",
		AccountName, tx.TransactionID)
	if c.GatewayURL != "" {
		paymentsURL = fmt.Sprintf("%s/api/payments/pub/transactions/%s/payments", c.GatewayURL, tx.TransactionID)
	}
	// deviceInfo is base64-encoded screen/device data for the VTEX payment gateway
	deviceInfo := "c3c9MTkyMCZzaD0xMDgwJmNkPTI0JnR6PTE4MCZsYW5nPXB0LUJSJmphdmE9ZmFsc2U="
	gatewayURL := fmt.Sprintf("%s?&orderId=%s&redirect=false&callbackUrl=%s&deviceInfo=%s&an=%s",
		paymentsURL, tx.OrderGroup, url.QueryEscape(callbackURL), deviceInfo, AccountName)

	_, err := c.PostJSONAbsolute(gatewayURL, payload)
	if err != nil {
		return fmt.Errorf("pay with saved card: %w", err)
	}
	return nil
}

func (c *Client) GatewayCallback(orderGroup string) error {
	path := fmt.Sprintf("/api/checkout/pub/gatewayCallback/%s", orderGroup)
	url := c.baseURL + path

	for attempt := range 10 {
		req, err := http.NewRequest("POST", url, http.NoBody)
		if err != nil {
			return fmt.Errorf("gateway callback: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if c.authToken != "" {
			req.AddCookie(&http.Cookie{
				Name:  "VtexIdclientAutCookie_" + AccountName,
				Value: c.authToken,
			})
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("gateway callback: %w", err)
		}
		cbBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == 200 || resp.StatusCode == 204 {
			return nil
		}
		if (resp.StatusCode == 428 || resp.StatusCode == 500) && attempt < 9 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		return fmt.Errorf("gateway callback: HTTP %d: %s", resp.StatusCode, string(cbBody))
	}
	return nil
}
