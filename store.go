// Package zonasul describes the Zona Sul supermarket storefront
// (www.zonasul.com.br) in Rio de Janeiro.
package zonasul

import (
	"github.com/voska/vtexkit/money"
	"github.com/voska/vtexkit/store"
)

// Store is the Zona Sul descriptor.
//
// Three things here cannot be discovered from the API, which is why this
// descriptor is larger than Frescatto's:
//
//   - MinOrder: Zona Sul enforces a R$100 floor that appears nowhere in
//     storePreferencesData; it surfaces only as a checkout error.
//   - OAuth: classic password auth is disabled on this account, so login
//     must be driven through their custom provider.
//   - Quirks: the payment gateway rejects card transactions without a
//     ClearSale device fingerprint (Cielo code 59), and needs the
//     gatewayCallback poll to settle a payment.
//
// Search is left at SearchAuto. This replaces the persisted GraphQL query
// the CLI used through v0.5.0, whose pinned hash VTEX rotates on every
// search-graphql release.
var Store = store.Store{
	Name:        "zonasul",
	DisplayName: "Zona Sul",
	BaseURL:     "https://www.zonasul.com.br",
	MinOrder:    money.Reais(100),
	OAuth:       Driver{},
	Quirks:      store.ClearSaleFingerprint | store.GatewayCallback,
}
