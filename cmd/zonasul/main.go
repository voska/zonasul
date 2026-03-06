package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/mattvoska/zonasul/internal/config"
	"github.com/mattvoska/zonasul/internal/exitcode"
	"github.com/mattvoska/zonasul/internal/outfmt"
	"github.com/mattvoska/zonasul/internal/vtex"
	"github.com/zalando/go-keyring"
)

const keyringService = "zonasul-cli"
const keyringUser = "vtex-jwt"

type Globals struct {
	JSON    bool `help:"Output JSON for agent consumption." env:"ZONASUL_JSON"`
	Plain   bool `help:"Output plain text for piping." env:"ZONASUL_PLAIN"`
	NoInput bool `help:"Disable interactive prompts." env:"ZONASUL_NO_INPUT"`
}

func (g *Globals) formatter() *outfmt.Formatter {
	return outfmt.FromGlobals(g.JSON, g.Plain)
}

func (g *Globals) client() *vtex.Client {
	token, _ := keyring.Get(keyringService, keyringUser)
	return vtex.NewClient(vtex.BaseURL, token)
}

func (g *Globals) authedClient() (*vtex.Client, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil || token == "" {
		return nil, fmt.Errorf("not logged in (run: zonasul auth login)")
	}
	return vtex.NewClient(vtex.BaseURL, token), nil
}

type AuthLoginCmd struct {
	Token string `help:"Provide JWT token directly (skip browser login)." env:"ZONASUL_TOKEN"`
}
type AuthStatusCmd struct{}
type AuthLogoutCmd struct{}

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Login via browser (Zona Sul OAuth)."`
	Status AuthStatusCmd `cmd:"" help:"Check auth state and token expiry."`
	Logout AuthLogoutCmd `cmd:"" help:"Clear stored credentials."`
}

type SearchCmd struct {
	Query string `arg:"" help:"Search query."`
	Limit int    `help:"Max results." default:"20"`
}

type CartAddCmd struct {
	SKU string `arg:"" help:"SKU ID to add."`
	Qty int    `help:"Quantity." default:"1"`
}

type CartRemoveCmd struct {
	Index int `arg:"" help:"Cart item index to remove."`
}

type CartClearCmd struct{}
type CartShowCmd struct{}

type CartCmd struct {
	Show   CartShowCmd   `cmd:"" default:"1" help:"Show current cart contents."`
	Add    CartAddCmd    `cmd:"" help:"Add item to cart."`
	Remove CartRemoveCmd `cmd:"" help:"Remove item from cart."`
	Clear  CartClearCmd  `cmd:"" help:"Clear all items from cart."`
}

type DeliveryWindowsCmd struct{}

type DeliveryCmd struct {
	Windows DeliveryWindowsCmd `cmd:"" help:"List available delivery windows."`
}

type CheckoutCmd struct {
	Window  int    `help:"Delivery window index." default:"-1"`
	Payment int    `help:"Payment method ID (ignored when saved card is used)." default:"-1"`
	CVV     string `help:"Credit card CVV for saved card payment." env:"ZONASUL_CVV"`
	Confirm bool   `help:"Actually place the order. Required safety gate."`
}

type AgentExitCodesCmd struct{}

type AgentCmd struct {
	ExitCodes AgentExitCodesCmd `cmd:"exit-codes" help:"Print exit code reference table."`
}

type OrdersCmd struct{}

type CLI struct {
	Globals

	Auth     AuthCmd     `cmd:"" help:"Authentication commands."`
	Search   SearchCmd   `cmd:"" help:"Search products."`
	Cart     CartCmd     `cmd:"" help:"Manage shopping cart."`
	Delivery DeliveryCmd `cmd:"" help:"Delivery options."`
	Checkout CheckoutCmd `cmd:"" help:"Place an order."`
	Orders   OrdersCmd   `cmd:"" help:"List recent orders."`
	Agent    AgentCmd    `cmd:"" help:"Agent introspection commands."`
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("zonasul"),
		kong.Description("Zona Sul supermarket CLI for AI agents."),
		kong.UsageOnError(),
	)

	err := ctx.Run(&cli.Globals)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.Error)
	}
}

func readLine(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func (c *AuthLoginCmd) Run(globals *Globals) error {
	var jwt string

	if c.Token != "" {
		jwt = c.Token
	} else if globals.NoInput {
		return fmt.Errorf("use --token or ZONASUL_TOKEN to provide a JWT in non-interactive mode")
	} else {
		fmt.Fprintln(os.Stderr, "Zona Sul uses custom OAuth login.")
		fmt.Fprintln(os.Stderr, "Choose login method:")
		fmt.Fprintln(os.Stderr, "  1) Paste JWT token from browser (DevTools > Application > Cookies > VtexIdclientAutCookie_zonasul)")
		fmt.Fprintln(os.Stderr, "  2) Open browser for OAuth login (experimental)")
		choice := readLine("Choice [1]: ")
		if choice == "2" {
			client := vtex.NewClient(vtex.BaseURL, "")
			var err error
			jwt, err = client.OAuthLogin()
			if err != nil {
				return err
			}
		} else {
			jwt = readLine("Paste JWT token: ")
			if jwt == "" {
				return fmt.Errorf("no token provided")
			}
		}
	}

	if err := keyring.Set(keyringService, keyringUser, jwt); err != nil {
		return fmt.Errorf("failed to store token in keychain: %w", err)
	}

	client := vtex.NewClient(vtex.BaseURL, jwt)
	user, err := client.AuthenticatedUser()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Warning: token may be invalid or expired")
	} else {
		fmt.Fprintf(os.Stderr, "Logged in as: %s\n", user)
	}

	fmt.Fprintln(os.Stderr, "Token stored in keychain.")

	if globals.JSON {
		return globals.formatter().Print(map[string]string{"status": "ok"})
	}
	return nil
}

func (c *AuthStatusCmd) Run(globals *Globals) error {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil || token == "" {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"status": "unauthenticated"})
		}
		fmt.Fprintln(os.Stderr, "Not logged in.")
		os.Exit(exitcode.AuthRequired)
	}

	client := vtex.NewClient(vtex.BaseURL, token)
	user, err := client.AuthenticatedUser()
	if err != nil {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"status": "expired"})
		}
		fmt.Fprintln(os.Stderr, "Token expired or invalid. Run: zonasul auth login")
		os.Exit(exitcode.AuthRequired)
	}

	if globals.JSON {
		return globals.formatter().Print(map[string]string{"status": "ok", "user": user})
	}
	fmt.Fprintf(os.Stderr, "Logged in as: %s\n", user)
	return nil
}

func (c *AuthLogoutCmd) Run(globals *Globals) error {
	_ = keyring.Delete(keyringService, keyringUser)
	fmt.Fprintln(os.Stderr, "Logged out. Token removed from keychain.")
	if globals.JSON {
		return globals.formatter().Print(map[string]string{"status": "logged_out"})
	}
	return nil
}

func (c *SearchCmd) Run(globals *Globals) error {
	client := globals.client()
	results, err := client.Search(c.Query, c.Limit)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		if globals.JSON {
			fmt.Println("[]")
		} else {
			fmt.Fprintln(os.Stderr, "No results found.")
		}
		os.Exit(exitcode.EmptyResults)
	}

	f := globals.formatter()
	if globals.JSON {
		return f.Print(results)
	}

	for i, r := range results {
		stock := ""
		if r.Available <= 0 {
			stock = " [out of stock]"
		}
		fmt.Printf("%-4d %-50s SKU:%-8s R$%.2f%s\n", i+1, r.Name, r.SKU, r.Price, stock)
	}
	return nil
}

func (c *CartShowCmd) Run(globals *Globals) error {
	client, err := globals.authedClient()
	if err != nil {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"error": "auth_required"})
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.AuthRequired)
		return nil
	}

	cfg, _ := config.Load(config.DefaultPath())
	of, err := client.GetOrderForm(cfg.OrderFormID)
	if err != nil {
		return err
	}

	cfg.OrderFormID = of.OrderFormID
	_ = config.Save(config.DefaultPath(), cfg)

	if globals.JSON {
		return globals.formatter().Print(of)
	}

	if len(of.Items) == 0 {
		fmt.Fprintln(os.Stderr, "Cart is empty.")
		os.Exit(exitcode.EmptyResults)
	}

	for i, item := range of.Items {
		fmt.Printf("%-4d %-50s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
	}
	for _, t := range of.Totalizers {
		fmt.Printf("%-55s R$%.2f\n", t.Name, float64(t.Value)/100)
	}
	return nil
}

func (c *CartAddCmd) Run(globals *Globals) error {
	client, err := globals.authedClient()
	if err != nil {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"error": "auth_required"})
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.AuthRequired)
		return nil
	}

	cfg, _ := config.Load(config.DefaultPath())
	of, err := client.AddToCart(cfg.OrderFormID, c.SKU, c.Qty)
	if err != nil {
		return err
	}

	cfg.OrderFormID = of.OrderFormID
	_ = config.Save(config.DefaultPath(), cfg)

	if globals.JSON {
		return globals.formatter().Print(of)
	}

	fmt.Fprintf(os.Stderr, "Added %s to cart.\n", c.SKU)
	for i, item := range of.Items {
		fmt.Printf("%-4d %-50s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
	}
	return nil
}

func (c *CartRemoveCmd) Run(globals *Globals) error {
	client, err := globals.authedClient()
	if err != nil {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"error": "auth_required"})
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.AuthRequired)
		return nil
	}

	cfg, _ := config.Load(config.DefaultPath())
	of, err := client.UpdateItemQuantity(cfg.OrderFormID, c.Index, 0)
	if err != nil {
		return err
	}

	if globals.JSON {
		return globals.formatter().Print(of)
	}
	fmt.Fprintf(os.Stderr, "Removed item %d from cart.\n", c.Index)
	return nil
}

func (c *CartClearCmd) Run(globals *Globals) error {
	client, err := globals.authedClient()
	if err != nil {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"error": "auth_required"})
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.AuthRequired)
		return nil
	}

	cfg, _ := config.Load(config.DefaultPath())
	if err := client.RemoveAllItems(cfg.OrderFormID); err != nil {
		return err
	}

	if globals.JSON {
		return globals.formatter().Print(map[string]string{"status": "cleared"})
	}
	fmt.Fprintln(os.Stderr, "Cart cleared.")
	return nil
}

func (c *DeliveryWindowsCmd) Run(globals *Globals) error {
	client, err := globals.authedClient()
	if err != nil {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"error": "auth_required"})
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.AuthRequired)
		return nil
	}

	cfg, _ := config.Load(config.DefaultPath())
	windows, err := client.GetDeliveryWindows(cfg.OrderFormID)
	if err != nil {
		return err
	}

	if len(windows) == 0 {
		if globals.JSON {
			fmt.Println("[]")
		} else {
			fmt.Fprintln(os.Stderr, "No delivery windows available.")
		}
		os.Exit(exitcode.EmptyResults)
	}

	if globals.JSON {
		return globals.formatter().Print(windows)
	}

	for i, w := range windows {
		price := "Grátis"
		if w.Price > 0 {
			price = fmt.Sprintf("R$%.2f", float64(w.Price)/100)
		}
		fmt.Printf("%-4d %s — %s  %s\n", i, w.Start.Local().Format("Mon 02 Jan 15:04"), w.End.Local().Format("15:04"), price)
	}
	return nil
}

func (c *CheckoutCmd) Run(globals *Globals) error {
	client, err := globals.authedClient()
	if err != nil {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"error": "auth_required"})
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.AuthRequired)
		return nil
	}

	cfg, _ := config.Load(config.DefaultPath())
	of, err := client.GetOrderForm(cfg.OrderFormID)
	if err != nil {
		return err
	}
	cfg.OrderFormID = of.OrderFormID
	_ = config.Save(config.DefaultPath(), cfg)

	if len(of.Items) == 0 {
		if globals.JSON {
			return globals.formatter().Print(map[string]string{"error": "empty_cart"})
		}
		fmt.Fprintln(os.Stderr, "Cart is empty. Add items first.")
		os.Exit(exitcode.EmptyResults)
		return nil
	}

	var itemsTotal int
	for _, t := range of.Totalizers {
		if t.ID == "Items" {
			itemsTotal = t.Value
		}
	}
	if itemsTotal < 10000 {
		if globals.JSON {
			return globals.formatter().Print(map[string]any{
				"error":    "min_order",
				"total":    itemsTotal,
				"required": 10000,
			})
		}
		fmt.Fprintf(os.Stderr, "Minimum order R$100.00, current total R$%.2f\n", float64(itemsTotal)/100)
		os.Exit(exitcode.MinOrder)
		return nil
	}

	// Set address first so delivery SLAs populate
	if err := client.SetAddress(of.OrderFormID, len(of.Items)); err != nil {
		return err
	}

	if c.Window >= 0 {
		windows, err := client.GetDeliveryWindows(of.OrderFormID)
		if err != nil {
			return err
		}
		if c.Window >= len(windows) {
			return fmt.Errorf("window index %d out of range (0-%d)", c.Window, len(windows)-1)
		}
		if err := client.SetShippingWindow(of.OrderFormID, windows[c.Window], len(of.Items)); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "Delivery window set.")
	}

	// Re-fetch orderForm to get updated total including shipping
	of, err = client.GetOrderForm(of.OrderFormID)
	if err != nil {
		return err
	}
	var orderTotal int
	for _, t := range of.Totalizers {
		orderTotal += t.Value
	}

	// Try to find saved card from any available orderForm
	savedCards, _ := client.GetSavedCards(of.OrderFormID)
	useCreditCard := c.CVV != "" && c.Payment < 0

	if useCreditCard {
		card := vtex.SavedCard{
			AccountID:         "AAAA1111BBBB2222CCCC3333DDDD4444",
			PaymentSystem:     "2",
			PaymentSystemName: "Visa",
			CardNumber:        "****9999",
		}
		if len(savedCards) > 0 {
			card = savedCards[0]
		}
		fmt.Fprintf(os.Stderr, "Using saved card: %s %s\n", card.PaymentSystemName, card.CardNumber)
		if err := client.SetPaymentWithSavedCard(of.OrderFormID, card, orderTotal); err != nil {
			return err
		}
	} else {
		paymentID := c.Payment
		if paymentID < 0 {
			paymentID = 125
		}
		if err := client.SetPayment(of.OrderFormID, paymentID, orderTotal); err != nil {
			return err
		}
	}

	if !c.Confirm {
		summary := map[string]any{
			"orderFormId": of.OrderFormID,
			"items":       of.Items,
			"totalizers":  of.Totalizers,
			"message":     "Use --confirm to place the order",
		}
		if globals.JSON {
			return globals.formatter().Print(summary)
		}
		fmt.Fprintln(os.Stderr, "Order summary:")
		for i, item := range of.Items {
			fmt.Printf("  %-4d %-45s x%-3d R$%.2f\n", i, item.Name, item.Quantity, float64(item.SellingPrice*item.Quantity)/100)
		}
		for _, t := range of.Totalizers {
			fmt.Printf("  %-50s R$%.2f\n", t.Name, float64(t.Value)/100)
		}
		fmt.Fprintln(os.Stderr, "\nUse --confirm to place the order.")
		return nil
	}

	tx, err := client.PlaceOrder(of.OrderFormID, orderTotal)
	if err != nil {
		return err
	}

	if useCreditCard {
		card := vtex.SavedCard{
			AccountID:         "AAAA1111BBBB2222CCCC3333DDDD4444",
			PaymentSystem:     "2",
			PaymentSystemName: "Visa",
			CardNumber:        "****9999",
		}
		if len(savedCards) > 0 {
			card = savedCards[0]
		}
		fmt.Fprintf(os.Stderr, "Processing credit card payment for %s %s...\n", card.PaymentSystemName, card.CardNumber)
		if err := client.PayWithSavedCard(tx, card, c.CVV, orderTotal); err != nil {
			return err
		}
		if err := client.GatewayCallback(tx.OrderGroup); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: gateway callback: %v\n", err)
		}
	}

	if globals.JSON {
		return globals.formatter().Print(map[string]string{"orderId": tx.OrderGroup, "status": "placed"})
	}
	fmt.Fprintf(os.Stderr, "Order placed! ID: %s\n", tx.OrderGroup)
	return nil
}

func (c *OrdersCmd) Run(globals *Globals) error {
	client, err := globals.authedClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.AuthRequired)
		return nil
	}

	orders, err := client.ListOrders()
	if err != nil {
		return err
	}

	if globals.JSON {
		return globals.formatter().Print(orders)
	}

	if len(orders) == 0 {
		fmt.Fprintln(os.Stderr, "No orders found.")
		os.Exit(exitcode.EmptyResults)
	}

	for _, o := range orders {
		fmt.Printf("%-20s  %-25s  R$%.2f  %s\n", o.OrderID, o.StatusDescription, float64(o.TotalValue)/100, o.CreationDate)
	}
	return nil
}

func (c *AgentExitCodesCmd) Run(globals *Globals) error {
	codes := []struct {
		Code int    `json:"code"`
		Name string `json:"name"`
	}{
		{0, "success"},
		{1, "error"},
		{2, "usage"},
		{3, "empty-results"},
		{4, "auth-required"},
		{5, "not-found"},
		{6, "min-order"},
		{7, "rate-limited"},
	}

	if globals.JSON {
		return globals.formatter().Print(codes)
	}

	fmt.Println("Code  Name")
	fmt.Println("----  ----")
	for _, c := range codes {
		fmt.Printf("%-6d%s\n", c.Code, c.Name)
	}
	return nil
}
