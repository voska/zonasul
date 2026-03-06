package vtex

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
)

type SearchResult struct {
	ProductID string  `json:"productId"`
	SKU       string  `json:"sku"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ListPrice float64 `json:"listPrice"`
	Available int     `json:"available"`
	Unit      string  `json:"unit"`
	UnitMult  float64 `json:"unitMultiplier"`
}

const searchHash = "31d3fa494df1fc41efef6d16dd96a96e6911b8aed7a037868699a1f3f4d365de"

func (c *Client) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}

	vars := map[string]any{
		"hideUnavailableItems":  true,
		"skusFilter":            "ALL",
		"simulationBehavior":    "default",
		"installmentCriteria":   "MAX_WITHOUT_INTEREST",
		"productOriginVtex":     true,
		"map":                   "ft",
		"query":                 query,
		"orderBy":               "OrderByScoreDESC",
		"from":                  0,
		"to":                    limit - 1,
		"selectedFacets":        []map[string]string{{"key": "ft", "value": query}},
		"fullText":              query,
		"facetsBehavior":        "Static",
		"categoryTreeBehavior":  "default",
		"withFacets":            false,
		"variant":               "null-null",
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
