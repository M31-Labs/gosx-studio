package sitehost

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
	"m31labs.dev/gosx-studio/cms/render"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// shop.go is the shop: products with pictures, variants, and stock; a shop
// page; a product block for any page; and a cart.
//
// Studio's library ships back-office renderers for products and orders but
// no model. The host keeps products beside the site data like forms, and
// the cart in a cookie that holds only product references and quantities —
// every price is looked up again at checkout, so nothing a visitor can edit
// changes what they pay.

const (
	shopPath         = "/shop"
	cartPath         = "/cart"
	productRefPrefix = "product:"
	currencyKey      = "currency"
	shopTitleKey     = "shopTitle"
	cartCookie       = "gosx_cart"
	productMaxImages = 6
	productMaxVars   = 20
	cartMaxLines     = 50
	cartMaxQty       = 99
)

// ProductImage is one picture of a product.
type ProductImage struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

// Variant is one option of a product: a size, a colour, a bundle.
type Variant struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	SKU   string `json:"sku,omitempty"`
	Price int64  `json:"price"` // minor units; 0 means the product's price
	Stock int    `json:"stock"`
}

// Product is one thing for sale. Prices are in the currency's minor unit.
type Product struct {
	ID          string         `json:"id"`
	Slug        string         `json:"slug"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Price       int64          `json:"price"`
	Compare     int64          `json:"compare,omitempty"` // a crossed-out "was" price
	Images      []ProductImage `json:"images,omitempty"`
	Variants    []Variant      `json:"variants,omitempty"`
	Stock       int            `json:"stock"`
	TrackStock  bool           `json:"trackStock"`
	Ships       bool           `json:"ships"`
	Active      bool           `json:"active"`
	Kind        string         `json:"kind,omitempty"`     // physical, digital, subscription, booking
	Interval    string         `json:"interval,omitempty"` // subscriptions: month, year, week
	File        string         `json:"file,omitempty"`     // digital: stored file name
	FileName    string         `json:"fileName,omitempty"` // digital: the name buyers see
	Booking     BookingRules   `json:"booking,omitempty"`  // booking: when it can be booked
	Created     time.Time      `json:"created"`
	Updated     time.Time      `json:"updated"`
}

func (p Product) ref() string { return productRefPrefix + p.ID }

func (p Product) path() string { return shopPath + "/" + p.Slug }

// variant finds one option by key; an empty key is "no option".
func (p Product) variant(key string) (Variant, bool) {
	key = strings.TrimSpace(key)
	for _, variant := range p.Variants {
		if variant.Key == key {
			return variant, true
		}
	}
	return Variant{}, false
}

// unitPrice is what one of this product, in this option, costs.
func (p Product) unitPrice(variantKey string) int64 {
	if variant, ok := p.variant(variantKey); ok && variant.Price > 0 {
		return variant.Price
	}
	return p.Price
}

// available is how many can be bought: -1 when stock is not tracked.
func (p Product) available(variantKey string) int {
	if !p.TrackStock {
		return -1
	}
	if variant, ok := p.variant(variantKey); ok {
		return variant.Stock
	}
	return p.Stock
}

func (p Product) soldOut() bool {
	if !p.TrackStock {
		return false
	}
	if len(p.Variants) == 0 {
		return p.Stock <= 0
	}
	for _, variant := range p.Variants {
		if variant.Stock > 0 {
			return false
		}
	}
	return true
}

// ---------- money ----------

var zeroDecimalCurrencies = map[string]bool{"jpy": true, "krw": true, "vnd": true, "clp": true, "isk": true, "huf": false}

var currencySymbols = map[string]string{"usd": "$", "eur": "€", "gbp": "£", "cad": "CA$", "aud": "A$", "nzd": "NZ$", "jpy": "¥", "chf": "CHF ", "sek": "kr ", "dkk": "kr ", "nok": "kr ", "pln": "zł ", "inr": "₹", "brl": "R$", "mxn": "MX$", "sgd": "S$", "hkd": "HK$", "zar": "R", "krw": "₩"}

// Currencies the shop can be priced in.
func Currencies() []string {
	return []string{"usd", "eur", "gbp", "cad", "aud", "nzd", "jpy", "chf", "sek", "dkk", "nok", "pln", "inr", "brl", "mxn", "sgd", "hkd", "zar", "krw"}
}

func normalizeCurrency(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	for _, known := range Currencies() {
		if known == code {
			return code
		}
	}
	return "usd"
}

// parseMoney reads what an owner types — "12", "12.5", "1,250.00", "£12" —
// into minor units.
func parseMoney(raw, currency string) (int64, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimLeft(raw, "$€£¥₹ ")
	raw = strings.ReplaceAll(raw, ",", "")
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 || math.IsNaN(value) || math.IsInf(value, 0) || value > 1e9 {
		return 0, errors.New("not a price")
	}
	if zeroDecimalCurrencies[normalizeCurrency(currency)] {
		return int64(math.Round(value)), nil
	}
	return int64(math.Round(value * 100)), nil
}

// moneyInput is the owner-facing text for a stored price.
func moneyInput(amount int64, currency string) string {
	if amount == 0 {
		return ""
	}
	if zeroDecimalCurrencies[normalizeCurrency(currency)] {
		return strconv.FormatInt(amount, 10)
	}
	return strconv.FormatInt(amount/100, 10) + "." + pad2int(int(amount%100))
}

func pad2int(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// formatMoney is the visitor-facing price.
func formatMoney(amount int64, currency string) string {
	currency = normalizeCurrency(currency)
	symbol, ok := currencySymbols[currency]
	if !ok {
		symbol = strings.ToUpper(currency) + " "
	}
	if zeroDecimalCurrencies[currency] {
		return symbol + groupThousands(amount)
	}
	return symbol + groupThousands(amount/100) + "." + pad2int(int(amount%100))
}

func groupThousands(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	first := len(s) % 3
	if first > 0 {
		b.WriteString(s[:first])
	}
	for i := first; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func (h *Host) currency() string { return normalizeCurrency(h.settings().Metadata[currencyKey]) }

func (h *Host) shopTitle() string {
	return firstNonEmpty(strings.TrimSpace(h.settings().Metadata[shopTitleKey]), "Shop")
}

// ---------- the store ----------

type productStore struct {
	mu       sync.Mutex
	path     string
	loaded   bool
	products []Product
}

func newProductStore(path string) *productStore { return &productStore{path: path} }

func (o Options) productsPath() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "products.json")
}

func (s *productStore) loadLocked() {
	if s.loaded {
		return
	}
	s.loaded = true
	if s.path == "" {
		return
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var file struct {
		Products []Product `json:"products"`
	}
	if json.Unmarshal(raw, &file) == nil {
		s.products = file.Products
	}
}

func (s *productStore) saveLocked() error {
	if s.path == "" {
		return nil
	}
	raw, err := json.MarshalIndent(struct {
		Products []Product `json:"products"`
	}{s.products}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, s.path)
}

func (s *productStore) list() []Product {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	out := make([]Product, len(s.products))
	copy(out, s.products)
	return out
}

func (s *productStore) get(id string) (Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, product := range s.products {
		if product.ID == id {
			return product, true
		}
	}
	return Product{}, false
}

func (s *productStore) bySlug(slug string) (Product, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for _, product := range s.products {
		if product.Slug == slug {
			return product, true
		}
	}
	return Product{}, false
}

var errProductNotFound = errors.New("product not found")

// put inserts or replaces a product, keeping its slug unique.
func (s *productStore) put(product Product) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	now := timeNow().UTC()
	product = normalizeProduct(product)
	base := firstNonEmpty(normalizeSlug(product.Slug), normalizeSlug(product.Name), "product")
	slug := base
	for n := 2; n < 1000; n++ {
		taken := false
		for _, other := range s.products {
			if other.Slug == slug && other.ID != product.ID {
				taken = true
				break
			}
		}
		if !taken {
			break
		}
		slug = base + "-" + strconv.Itoa(n)
	}
	product.Slug = slug
	product.Updated = now
	if product.ID == "" {
		product.ID = "p" + randomHex(4)
		product.Created = now
		s.products = append(s.products, product)
		return product, s.saveLocked()
	}
	for index, existing := range s.products {
		if existing.ID == product.ID {
			product.Created = existing.Created
			s.products[index] = product
			return product, s.saveLocked()
		}
	}
	return Product{}, errProductNotFound
}

// adjustStock takes quantity off a product or variant; it never goes below
// zero and never touches untracked stock.
func (s *productStore) adjustStock(id, variantKey string, delta int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index := range s.products {
		product := &s.products[index]
		if product.ID != id || !product.TrackStock {
			continue
		}
		if variantKey == "" {
			product.Stock = max0(product.Stock + delta)
		} else {
			for v := range product.Variants {
				if product.Variants[v].Key == variantKey {
					product.Variants[v].Stock = max0(product.Variants[v].Stock + delta)
				}
			}
		}
		product.Updated = timeNow().UTC()
		return s.saveLocked()
	}
	return nil
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func (s *productStore) remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
	for index, existing := range s.products {
		if existing.ID == id {
			s.products = append(s.products[:index], s.products[index+1:]...)
			return s.saveLocked()
		}
	}
	return errProductNotFound
}

func normalizeProduct(product Product) Product {
	product.Name = firstNonEmpty(strings.TrimSpace(product.Name), "Product")
	product.Kind = normalizeKind(product.Kind)
	if product.Kind != kindPhysical {
		product.Ships = false
		product.TrackStock = product.Kind == kindBooking && false
	}
	if product.Kind == kindSubscription {
		product.Interval = normalizeInterval(product.Interval)
	} else {
		product.Interval = ""
	}
	if product.Kind != kindDigital {
		product.File, product.FileName = "", ""
	}
	if product.Kind == kindBooking {
		product.Booking = normalizeBookingRules(product.Booking)
	} else {
		product.Booking = BookingRules{}
	}
	product.Description = strings.TrimSpace(product.Description)
	if product.Price < 0 {
		product.Price = 0
	}
	if product.Compare <= product.Price {
		product.Compare = 0
	}
	images := make([]ProductImage, 0, len(product.Images))
	for _, image := range product.Images {
		if url := strings.TrimSpace(image.URL); url != "" && len(images) < productMaxImages {
			images = append(images, ProductImage{URL: url, Alt: strings.TrimSpace(image.Alt)})
		}
	}
	product.Images = images
	variants := make([]Variant, 0, len(product.Variants))
	seen := map[string]bool{}
	for index, variant := range product.Variants {
		variant.Name = strings.TrimSpace(variant.Name)
		if variant.Name == "" {
			continue
		}
		key := normalizeSlug(variant.Name)
		if key == "" || seen[key] {
			key = key + "-" + strconv.Itoa(index+1)
		}
		seen[key] = true
		variant.Key = key
		variant.SKU = strings.TrimSpace(variant.SKU)
		if variant.Price < 0 {
			variant.Price = 0
		}
		variant.Stock = max0(variant.Stock)
		variants = append(variants, variant)
		if len(variants) == productMaxVars {
			break
		}
	}
	product.Variants = variants
	product.Stock = max0(product.Stock)
	return product
}

// activeProducts is what the shop shows, newest first.
func (h *Host) activeProducts() []Product {
	all := h.products.list()
	out := make([]Product, 0, len(all))
	for _, product := range all {
		if product.Active {
			out = append(out, product)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Created.After(out[j].Created) })
	return out
}

func (h *Host) shopInMenu() bool { return len(h.activeProducts()) > 0 }

// productByRef resolves a block's reference: "product:<id>", an id, or a slug.
func (h *Host) productByRef(ref string) (Product, bool) {
	ref = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ref), productRefPrefix))
	if ref == "" {
		return Product{}, false
	}
	if product, ok := h.products.get(ref); ok {
		return product, true
	}
	return h.products.bySlug(ref)
}

// ---------- routes ----------

func (h *Host) mountShop(mux *http.ServeMux) {
	h.mountGoods(mux)
	h.mountBookings(mux)
	mux.HandleFunc("GET "+shopPath, h.handleShop)
	mux.HandleFunc("GET "+shopPath+"/{$}", h.handleShop)
	mux.HandleFunc("GET "+shopPath+"/{slug}", h.handleProductPage)
	mux.HandleFunc("GET "+cartPath, h.handleCart)
	mux.HandleFunc("GET "+cartPath+"/{$}", h.handleCart)
	mux.HandleFunc("POST "+cartPath+"/add", h.handleCartAdd)
	mux.HandleFunc("POST "+cartPath+"/update", h.handleCartUpdate)

	mux.HandleFunc("GET /admin/shop", h.handleAdminShop)
	mux.HandleFunc("GET /admin/shop/{$}", h.handleAdminShop)
	mux.HandleFunc("POST /admin/shop", h.handleAdminCreateProduct)
	mux.HandleFunc("POST /admin/shop/{$}", h.handleAdminCreateProduct)
	mux.HandleFunc("GET /admin/shop/{id}", h.handleAdminProduct)
	mux.HandleFunc("POST /admin/shop/{id}", h.handleAdminProductAction)
	mux.HandleFunc("POST /admin/shop/{id}/delete", h.handleAdminProductDelete)
}

// ---------- public: shop and product pages ----------

func (h *Host) publicShell(r *http.Request, active string, meta PageMeta, main gosx.Node) (PageMeta, gosx.Node) {
	settings := h.settings()
	meta.HeadCode = h.headCode()
	meta.Consent = h.consentRequired()
	meta.Feed = h.blogInMenu()
	var consent gosx.Node = gosx.Fragment()
	if meta.HeadCode != "" && meta.Consent {
		consent = renderConsentBanner()
	}
	shell := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-shell")),
		h.renderPublicNav(settings, active),
		gosx.El("main", gosx.Attrs(gosx.Attr("class", "site-main"), gosx.Attr("id", "main")), main),
		h.renderPublicFooter(settings),
		consent,
	)
	return meta, shell
}

func (h *Host) handleShop(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	products := h.activeProducts()
	meta := metaFromSettings(settings)
	meta.Title = h.shopTitle()
	meta.Description = "Things for sale from " + firstNonEmpty(settings.Title, h.opts.SiteTitle) + "."
	meta.CanonicalPath = shopPath
	meta.Kind = "website"
	status := http.StatusOK
	var body gosx.Node
	if len(products) == 0 {
		status = http.StatusNotFound
		meta.NoIndex = true
		body = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("Nothing for sale yet. Check back soon."))
	} else {
		body = h.renderProductGrid(products)
	}
	meta, shell := h.publicShell(r, "shop", meta, gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-article site-shop")),
		gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text(h.shopTitle())),
		body,
	))
	h.writeDocument(w, status, meta, shell)
}

func (h *Host) renderProductGrid(products []Product) gosx.Node {
	cards := make([]gosx.Node, 0, len(products))
	for _, product := range products {
		cards = append(cards, h.renderProductCard(product, false))
	}
	return gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-products")), gosx.Fragment(cards...))
}

// renderProductCard is a product in a grid or on a page: picture, name,
// price, and the way in.
func (h *Host) renderProductCard(product Product, inline bool) gosx.Node {
	currency := h.currency()
	var picture gosx.Node = gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-product-card__blank")))
	if len(product.Images) > 0 {
		if attrs, ok := h.imageAttrs(product.Images[0].URL, firstNonEmpty(product.Images[0].Alt, product.Name), gallerySizes); ok {
			picture = gosx.El("img", gosx.Attrs(attrs...))
		}
	}
	price := []gosx.Node{gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-price")), gosx.Text(priceLabel(product, currency)))}
	if product.Compare > 0 {
		price = append(price, gosx.Text(" "), gosx.El("s", gosx.Attrs(gosx.Attr("class", "site-price--was")), gosx.Text(formatMoney(product.Compare, currency))))
	}
	if product.soldOut() {
		price = append(price, gosx.Text(" "), gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-badge")), gosx.Text("Sold out")))
	}
	class := "site-product-card"
	if inline {
		class += " site-product-card--inline"
	}
	return gosx.El("li", gosx.Attrs(gosx.Attr("class", class)),
		gosx.El("a", gosx.Attrs(gosx.Attr("class", "site-product-card__link"), gosx.Attr("href", product.path())),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-product-card__picture")), picture),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-product-card__name")), gosx.Text(product.Name)),
			gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-product-card__price")), gosx.Fragment(price...)),
		),
	)
}

// productHook renders the product block on any page.
func (h *Host) productHook() render.Hook {
	return func(ctx render.Context) (gosx.Node, bool) {
		product, ok := h.productByRef(ctx.Ref)
		if !ok || !product.Active {
			return gosx.Fragment(), true
		}
		return gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-products site-products--single")), h.renderProductCard(product, true)), true
	}
}

func (h *Host) hooksFor(pagePath string, state formState) render.Hooks {
	return render.Hooks{Flow: h.flowHook(pagePath, state), Image: h.imageHook(), Product: h.productHook()}
}

func (h *Host) handleProductPage(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	product, ok := h.products.bySlug(strings.TrimSpace(r.PathValue("slug")))
	if !ok || !product.Active {
		h.servePublicNotFound(w, settings, "shop/"+r.PathValue("slug"))
		return
	}
	currency := h.currency()
	brand := brandFromSettings(settings)
	meta := metaFromSettings(settings)
	meta.Title = product.Name
	meta.Description = firstNonEmpty(shorten(inlineToPlain(product.Description), excerptLength), settings.Description)
	meta.CanonicalPath = product.path()
	meta.Kind = "product"
	if len(product.Images) > 0 {
		meta.ImageURL = product.Images[0].URL
		meta.ImageAlt = firstNonEmpty(product.Images[0].Alt, product.Name)
	} else {
		meta.ImageURL = brand.LogoURL
	}
	meta.JSONLD = h.productData(settings, product, h.absoluteBase(r))

	pictures := make([]gosx.Node, 0, len(product.Images))
	for index, image := range product.Images {
		sizes := "(max-width: 720px) 100vw, 480px"
		if index > 0 {
			sizes = gallerySizes
		}
		if attrs, ok := h.imageAttrs(image.URL, firstNonEmpty(image.Alt, product.Name), sizes); ok {
			pictures = append(pictures, gosx.El("figure", gosx.Attrs(gosx.Attr("class", "site-product__picture")), gosx.El("img", gosx.Attrs(attrs...))))
		}
	}

	price := []gosx.Node{gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-price site-price--big")), gosx.Text(priceLabel(product, currency)))}
	if product.Compare > 0 {
		price = append(price, gosx.Text(" "), gosx.El("s", gosx.Attrs(gosx.Attr("class", "site-price--was")), gosx.Text(formatMoney(product.Compare, currency))))
	}

	message := r.URL.Query().Get("added")
	var notice gosx.Node = gosx.Fragment()
	if message == "1" {
		notice = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-notice"), gosx.Attr("role", "status")),
			gosx.Text("Added to your cart. "), gosx.El("a", gosx.Attrs(gosx.Attr("href", cartPath)), gosx.Text("View cart")))
	}

	var bookingDay *time.Time
	if product.Kind == kindBooking {
		if day, err := time.ParseInLocation("2006-01-02", r.URL.Query().Get("date"), time.Local); err == nil {
			bookingDay = &day
		}
		switch r.URL.Query().Get("problem") {
		case "slot":
			notice = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "alert")), gosx.Text("That time isn't free any more, or a name was missing. Pick again."))
		case "email":
			notice = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "alert")), gosx.Text("That email address doesn't look right."))
		}
	}
	details := gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-product__details")),
		gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text(product.Name)),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-product__price")), gosx.Fragment(price...)),
		notice,
		h.renderBuyForm(product, bookingDay),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-product__description")), renderInline(product.Description)),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-nav")), gosx.El("a", gosx.Attrs(gosx.Attr("href", shopPath)), gosx.Text("← Back to the shop"))),
	)
	main := gosx.El("article", gosx.Attrs(gosx.Attr("class", "site-product")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-product__pictures")), gosx.Fragment(pictures...)),
		details,
	)
	meta, shell := h.publicShell(r, "shop", meta, main)
	h.writeDocument(w, http.StatusOK, meta, shell)
}

// renderBuyForm is the option picker, quantity, and Add to cart button.
func (h *Host) renderBuyForm(product Product, bookingDay *time.Time) gosx.Node {
	currency := h.currency()
	if product.soldOut() {
		return gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-product__stock")), gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-badge")), gosx.Text("Sold out")))
	}
	nodes := []gosx.Node{}
	if len(product.Variants) > 0 {
		options := make([]gosx.Node, 0, len(product.Variants))
		for _, variant := range product.Variants {
			label := variant.Name
			if variant.Price > 0 && variant.Price != product.Price {
				label += " — " + formatMoney(variant.Price, currency)
			}
			attrs := []any{gosx.Attr("value", variant.Key)}
			if product.TrackStock && variant.Stock <= 0 {
				attrs = append(attrs, gosx.Attr("disabled", "disabled"))
				label += " (sold out)"
			}
			options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(label)))
		}
		nodes = append(nodes, gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field")),
			gosx.El("span", nil, gosx.Text("Option")),
			gosx.El("select", gosx.Attrs(gosx.Attr("name", "variant"), gosx.Attr("required", "required")), gosx.Fragment(options...))))
	}
	stockNote := gosx.Fragment()
	if product.TrackStock && len(product.Variants) == 0 && product.Stock <= 5 {
		stockNote = gosx.El("span", gosx.Attrs(gosx.Attr("class", "site-product__stock")), gosx.Text("Only "+strconv.Itoa(product.Stock)+" left"))
	}
	buttonText := "Add to cart"
	var qtyField gosx.Node = gosx.El("label", gosx.Attrs(gosx.Attr("class", "site-form__field site-buy__qty")),
		gosx.El("span", nil, gosx.Text("Quantity")),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "number"), gosx.Attr("name", "qty"), gosx.Attr("value", "1"), gosx.Attr("min", "1"), gosx.Attr("max", strconv.Itoa(cartMaxQty)), gosx.Attr("inputmode", "numeric"))))
	switch product.Kind {
	case kindDigital:
		buttonText = "Buy"
		qtyField = gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "qty"), gosx.Attr("value", "1")))
	case kindSubscription:
		buttonText = "Subscribe"
		qtyField = gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "qty"), gosx.Attr("value", "1")))
	case kindBooking:
		return h.renderBookingForm(product, bookingDay)
	}
	nodes = append(nodes,
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-buy__row")),
			qtyField,
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("type", "submit")), gosx.Text(buttonText)),
		),
		stockNote,
	)
	return gosx.El("form", gosx.Attrs(gosx.Attr("class", "site-form site-buy"), gosx.Attr("method", "post"), gosx.Attr("action", cartPath+"/add")),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "product"), gosx.Attr("value", product.ID))),
		gosx.Fragment(nodes...),
	)
}

func (h *Host) productData(settings cmsstore.SiteSettings, product Product, base string) []map[string]any {
	offer := map[string]any{
		"@type":         "Offer",
		"url":           base + product.path(),
		"priceCurrency": strings.ToUpper(h.currency()),
		"price":         moneyInput(product.Price, h.currency()),
		"availability":  "https://schema.org/InStock",
	}
	if product.Price == 0 {
		offer["price"] = "0"
	}
	if product.soldOut() {
		offer["availability"] = "https://schema.org/OutOfStock"
	}
	data := map[string]any{
		"@context": "https://schema.org",
		"@type":    "Product",
		"name":     product.Name,
		"url":      base + product.path(),
		"offers":   offer,
	}
	if description := inlineToPlain(product.Description); description != "" {
		data["description"] = description
	}
	if len(product.Images) > 0 {
		images := make([]string, 0, len(product.Images))
		for _, image := range product.Images {
			images = append(images, absoluteURL(base, image.URL))
		}
		data["image"] = images
	}
	return []map[string]any{data}
}

// ---------- the cart ----------

type cartLine struct {
	Product string `json:"p"`
	Variant string `json:"v,omitempty"`
	Qty     int    `json:"q"`
	Slot    string `json:"s,omitempty"` // bookings: the chosen start time, RFC 3339
}

// resolvedLine is a cart line with its product looked up and its quantity
// checked against stock.
type resolvedLine struct {
	Line    cartLine
	Product Product
	Variant Variant
	Unit    int64
	Total   int64
	Name    string
	Capped  bool
}

func readCart(r *http.Request) []cartLine {
	cookie, err := r.Cookie(cartCookie)
	if err != nil || cookie.Value == "" {
		return nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil
	}
	var lines []cartLine
	if json.Unmarshal(raw, &lines) != nil {
		return nil
	}
	if len(lines) > cartMaxLines {
		lines = lines[:cartMaxLines]
	}
	return lines
}

func writeCart(w http.ResponseWriter, r *http.Request, lines []cartLine) {
	cookie := &http.Cookie{Name: cartCookie, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")}
	if len(lines) == 0 {
		cookie.MaxAge = -1
	} else {
		raw, _ := json.Marshal(lines)
		cookie.Value = base64.RawURLEncoding.EncodeToString(raw)
		cookie.MaxAge = 30 * 24 * 3600
	}
	http.SetCookie(w, cookie)
}

// resolveCart drops lines whose product is gone or inactive and caps
// quantities at what is in stock.
func (h *Host) resolveCart(lines []cartLine) ([]resolvedLine, int64) {
	out := make([]resolvedLine, 0, len(lines))
	var subtotal int64
	for _, line := range lines {
		product, ok := h.products.get(line.Product)
		if !ok || !product.Active {
			continue
		}
		variant, hasVariant := product.variant(line.Variant)
		if len(product.Variants) > 0 && !hasVariant {
			continue
		}
		qty := line.Qty
		if qty < 1 {
			continue
		}
		if qty > cartMaxQty {
			qty = cartMaxQty
		}
		if product.Kind == kindSubscription || product.Kind == kindDigital || product.Kind == kindBooking {
			qty = 1
		}
		if product.Kind == kindBooking {
			start, err := time.Parse(time.RFC3339, line.Slot)
			if err != nil || !h.slotOK(product, start) {
				continue
			}
		}
		capped := false
		if available := product.available(line.Variant); available >= 0 && qty > available {
			qty, capped = available, true
		}
		if qty == 0 {
			continue
		}
		unit := product.unitPrice(line.Variant)
		name := product.Name
		if hasVariant {
			name += " — " + variant.Name
		}
		if product.Kind == kindSubscription {
			name += " (per " + normalizeInterval(product.Interval) + ")"
		}
		if line.Slot != "" {
			name += " — " + slotLabel(line.Slot)
		}
		line.Qty = qty
		out = append(out, resolvedLine{Line: line, Product: product, Variant: variant, Unit: unit, Total: unit * int64(qty), Name: name, Capped: capped})
		subtotal += unit * int64(qty)
	}
	return out, subtotal
}

func anyCapped(resolved []resolvedLine) bool {
	for _, line := range resolved {
		if line.Capped {
			return true
		}
	}
	return false
}

func linesOf(resolved []resolvedLine) []cartLine {
	out := make([]cartLine, 0, len(resolved))
	for _, line := range resolved {
		out = append(out, line.Line)
	}
	return out
}

func (h *Host) handleCartAdd(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	product, ok := h.products.get(strings.TrimSpace(r.PostFormValue("product")))
	if !ok || !product.Active {
		http.Redirect(w, r, shopPath, http.StatusSeeOther)
		return
	}
	qty, _ := strconv.Atoi(strings.TrimSpace(r.PostFormValue("qty")))
	if qty < 1 {
		qty = 1
	}
	variant := strings.TrimSpace(r.PostFormValue("variant"))
	if len(product.Variants) > 0 {
		if _, ok := product.variant(variant); !ok {
			http.Redirect(w, r, product.path(), http.StatusSeeOther)
			return
		}
	} else {
		variant = ""
	}
	slot := ""
	if product.Kind == kindBooking {
		start, err := time.Parse(time.RFC3339, r.PostFormValue("slot"))
		if err != nil || !h.slotOK(product, start) {
			http.Redirect(w, r, product.path()+"?problem=slot", http.StatusSeeOther)
			return
		}
		slot = start.UTC().Format(time.RFC3339)
		qty = 1
	}
	lines := readCart(r)
	merged := false
	for index := range lines {
		if lines[index].Product == product.ID && lines[index].Variant == variant && slot == "" && lines[index].Slot == "" {
			lines[index].Qty += qty
			merged = true
			break
		}
	}
	if !merged {
		lines = append(lines, cartLine{Product: product.ID, Variant: variant, Qty: qty, Slot: slot})
	}
	// The quantity is kept as asked; the cart page trims it to stock and
	// says so, once, where the visitor can see it.
	for index := range lines {
		if lines[index].Qty > cartMaxQty {
			lines[index].Qty = cartMaxQty
		}
	}
	if len(lines) > cartMaxLines {
		lines = lines[len(lines)-cartMaxLines:]
	}
	writeCart(w, r, lines)
	http.Redirect(w, r, product.path()+"?added=1", http.StatusSeeOther)
}

func (h *Host) handleCartUpdate(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	lines := readCart(r)
	for index := range lines {
		if raw := r.PostFormValue("qty_" + strconv.Itoa(index)); raw != "" {
			if qty, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
				lines[index].Qty = qty
			}
		}
		if r.PostFormValue("remove") == strconv.Itoa(index) {
			lines[index].Qty = 0
		}
	}
	resolved, _ := h.resolveCart(lines)
	writeCart(w, r, linesOf(resolved))
	http.Redirect(w, r, cartPath, http.StatusSeeOther)
}

func (h *Host) handleCart(w http.ResponseWriter, r *http.Request) {
	settings := h.settings()
	raw := readCart(r)
	resolved, subtotal := h.resolveCart(raw)
	// Whatever was trimmed or dropped is written back, so the note below
	// shows once and the next view is clean.
	if normalized := linesOf(resolved); len(normalized) != len(raw) || anyCapped(resolved) {
		writeCart(w, r, normalized)
	}
	currency := h.currency()
	meta := metaFromSettings(settings)
	meta.Title = "Your cart"
	meta.NoIndex = true
	meta.CanonicalPath = cartPath

	var failed gosx.Node = gosx.Fragment()
	if r.URL.Query().Get("checkout") == "failed" {
		failed = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "alert")), gosx.Text("We couldn't start the payment. Please try again in a moment."))
	}
	var body gosx.Node
	if len(resolved) == 0 {
		body = gosx.Fragment(
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-lede")), gosx.Text("Your cart is empty.")),
			gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("href", shopPath)), gosx.Text("Go to the shop"))),
		)
	} else {
		rows := make([]gosx.Node, 0, len(resolved))
		capped := false
		for index, line := range resolved {
			capped = capped || line.Capped
			n := strconv.Itoa(index)
			var picture gosx.Node = gosx.Fragment()
			if len(line.Product.Images) > 0 {
				if attrs, ok := h.imageAttrs(line.Product.Images[0].URL, line.Product.Name, "80px"); ok {
					picture = gosx.El("img", gosx.Attrs(append(attrs, gosx.Attr("class", "site-cart__thumb"))...))
				}
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "site-cart__item")), picture,
					gosx.El("a", gosx.Attrs(gosx.Attr("href", line.Product.path())), gosx.Text(line.Name))),
				gosx.El("td", nil, gosx.Text(formatMoney(line.Unit, currency))),
				gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("class", "site-cart__qty"), gosx.Attr("type", "number"), gosx.Attr("name", "qty_"+n), gosx.Attr("value", strconv.Itoa(line.Line.Qty)), gosx.Attr("min", "0"), gosx.Attr("max", strconv.Itoa(cartMaxQty)), gosx.Attr("aria-label", "Quantity of "+line.Name)))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "site-cart__total")), gosx.Text(formatMoney(line.Total, currency))),
				gosx.El("td", nil, gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-cart__remove"), gosx.Attr("type", "submit"), gosx.Attr("name", "remove"), gosx.Attr("value", n), gosx.Attr("aria-label", "Remove "+line.Name)), gosx.Text("✕"))),
			))
		}
		var cappedNote gosx.Node = gosx.Fragment()
		if capped {
			cappedNote = gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-form__error"), gosx.Attr("role", "status")), gosx.Text("We've adjusted a quantity to what's in stock."))
		}
		body = gosx.El("form", gosx.Attrs(gosx.Attr("class", "site-cart"), gosx.Attr("method", "post"), gosx.Attr("action", cartPath+"/update")),
			cappedNote,
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "site-cart__table")),
				gosx.El("thead", nil, gosx.El("tr", nil,
					gosx.El("th", nil, gosx.Text("Item")), gosx.El("th", nil, gosx.Text("Price")), gosx.El("th", nil, gosx.Text("Quantity")), gosx.El("th", nil, gosx.Text("Total")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...)),
				gosx.El("tfoot", nil, gosx.El("tr", nil,
					gosx.El("td", gosx.Attrs(gosx.Attr("colspan", "3")), gosx.Text("Subtotal")),
					gosx.El("td", gosx.Attrs(gosx.Attr("class", "site-cart__total")), gosx.Text(formatMoney(subtotal, currency))),
					gosx.El("td", nil))),
			),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-cart__actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-cart__update"), gosx.Attr("type", "submit")), gosx.Text("Update cart")),
				h.renderCheckoutButton(resolved),
			),
		)
	}
	meta, shell := h.publicShell(r, "cart", meta, gosx.El("section", gosx.Attrs(gosx.Attr("class", "site-article site-cart-page")),
		gosx.El("h1", gosx.Attrs(gosx.Attr("class", "site-title")), gosx.Text("Your cart")),
		failed,
		body,
	))
	h.writeDocument(w, http.StatusOK, meta, shell)
}

// renderCheckoutButton is filled in by checkout.go; until payments are
// connected it says so.
func (h *Host) renderCheckoutButton(resolved []resolvedLine) gosx.Node {
	if !h.checkoutReady() {
		return gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-cart__note")), gosx.Text("Checkout isn't open yet."))
	}
	return gosx.El("button", gosx.Attrs(gosx.Attr("class", "site-button"), gosx.Attr("type", "submit"), gosx.Attr("formaction", checkoutPath), gosx.Attr("formmethod", "post")), gosx.Text("Checkout"))
}
