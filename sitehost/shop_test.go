package sitehost

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMoneyParsesAndFormats(t *testing.T) {
	for in, want := range map[string]int64{"12.50": 1250, "12": 1200, "1,250.00": 125000, "£9.99": 999, "": 0, "0.1": 10} {
		got, err := parseMoney(in, "gbp")
		if err != nil || got != want {
			t.Fatalf("parseMoney(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := parseMoney("free", "usd"); err == nil {
		t.Fatal("words are not prices")
	}
	if got, _ := parseMoney("1200", "jpy"); got != 1200 {
		t.Fatalf("yen has no minor unit: %d", got)
	}
	for _, c := range []struct {
		amount   int64
		currency string
		want     string
	}{{1250, "usd", "$12.50"}, {125000, "eur", "€1,250.00"}, {999, "gbp", "£9.99"}, {1200, "jpy", "¥1,200"}, {5, "usd", "$0.05"}} {
		if got := formatMoney(c.amount, c.currency); got != c.want {
			t.Fatalf("formatMoney(%d, %s) = %q, want %q", c.amount, c.currency, got, c.want)
		}
	}
	if moneyInput(1250, "usd") != "12.50" || moneyInput(0, "usd") != "" {
		t.Fatal("moneyInput round trip")
	}
}

func createProduct(t *testing.T, handler http.Handler, name, price string) string {
	t.Helper()
	rec := post(t, handler, "/admin/shop", url.Values{"name": {name}, "price": {price}})
	location := rec.Header().Get("Location")
	if rec.Code != http.StatusSeeOther || !strings.HasPrefix(location, "/admin/shop/") {
		t.Fatalf("create product = %d %q %s", rec.Code, location, rec.Body.String())
	}
	return strings.TrimPrefix(strings.Split(location, "?")[0], "/admin/shop/")
}

// saveProduct posts the editor form as the browser would.
func saveProduct(t *testing.T, handler http.Handler, id string, fields map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{}
	for key, value := range fields {
		form.Set(key, value)
	}
	form.Set("save", "1")
	return postMultipart(t, handler, "/admin/shop/"+id, form)
}

func TestShopFromFirstProductToCart(t *testing.T) {
	host, handler := newTestHost(t)
	if strings.Contains(siteNav(t, handler), `href="/shop"`) {
		t.Fatal("no Shop in the menu before a product is on sale")
	}
	if code := get(t, handler, "/shop").Code; code != http.StatusNotFound {
		t.Fatalf("/shop with nothing = %d", code)
	}

	id := createProduct(t, handler, "Sourdough loaf", "6.50")
	product, _ := host.products.get(id)
	if product.Slug != "sourdough-loaf" || product.Price != 650 || product.Active {
		t.Fatalf("new product: %+v", product)
	}
	// Not on sale yet: invisible to visitors.
	if code := get(t, handler, "/shop/sourdough-loaf").Code; code != http.StatusNotFound {
		t.Fatal("a product that is not on sale must not be served")
	}

	imgURL := uploadedURL(t, postUpload(t, handler, "loaf.png", bigPNG(t, 1200, 900)).Body.String())
	rec := saveProduct(t, handler, id, map[string]string{
		"name": "Sourdough loaf", "slug": "sourdough-loaf", "description": "Slow-fermented, **crusty**, 800g.",
		"price": "6.50", "compare": "8.00", "imageCount": "0", "newImageURL": imgURL,
		"variantCount": "0", "trackStock": "1", "stock": "3", "ships": "1", "active": "1",
	})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("save = %d %s", rec.Code, rec.Body.String())
	}
	// Add two options with their own stock.
	postMultipart(t, handler, "/admin/shop/"+id, url.Values{"name": {"Sourdough loaf"}, "price": {"6.50"}, "compare": {"8.00"}, "description": {"Slow-fermented, **crusty**, 800g."}, "imageCount": {"1"}, "image_0": {imgURL}, "alt_0": {"A loaf"}, "variantCount": {"0"}, "trackStock": {"1"}, "active": {"1"}, "ships": {"1"}, "addVariant": {"1"}})
	postMultipart(t, handler, "/admin/shop/"+id, url.Values{"name": {"Sourdough loaf"}, "price": {"6.50"}, "compare": {"8.00"}, "description": {"Slow-fermented, **crusty**, 800g."}, "imageCount": {"1"}, "image_0": {imgURL}, "alt_0": {"A loaf"},
		"variantCount": {"1"}, "vname_0": {"Small"}, "vprice_0": {""}, "vstock_0": {"2"}, "trackStock": {"1"}, "active": {"1"}, "ships": {"1"}, "addVariant": {"1"}})
	postMultipart(t, handler, "/admin/shop/"+id, url.Values{"name": {"Sourdough loaf"}, "price": {"6.50"}, "compare": {"8.00"}, "description": {"Slow-fermented, **crusty**, 800g."}, "imageCount": {"1"}, "image_0": {imgURL}, "alt_0": {"A loaf"},
		"variantCount": {"2"}, "vname_0": {"Small"}, "vprice_0": {""}, "vstock_0": {"2"}, "vname_1": {"Large"}, "vprice_1": {"9.00"}, "vstock_1": {"0"}, "trackStock": {"1"}, "active": {"1"}, "ships": {"1"}, "save": {"1"}})
	product, _ = host.products.get(id)
	if len(product.Variants) != 2 || product.Variants[0].Key != "small" || product.Variants[1].Price != 900 || product.Variants[1].Stock != 0 {
		t.Fatalf("variants: %+v", product.Variants)
	}

	// The shop and the product page.
	mustContain(t, siteNav(t, handler), `href="/shop">Shop</a>`, "a product on sale puts Shop in the menu")
	shop := get(t, handler, "/shop").Body.String()
	mustContain(t, shop, `href="/shop/sourdough-loaf"`, "the shop lists the product")
	mustContain(t, shop, `<span class="site-price">$6.50</span>`, "with its price")
	page := get(t, handler, "/shop/sourdough-loaf").Body.String()
	mustContain(t, page, "Slow-fermented, <strong>crusty</strong>, 800g.", "the description keeps formatting")
	mustContain(t, page, `<s class="site-price--was">$8.00</s>`, "the was price is crossed out")
	mustContain(t, page, `<option value="small">Small</option>`, "options are offered")
	mustContain(t, page, `<option value="large" disabled="disabled">Large — $9.00 (sold out)</option>`, "a sold-out option is disabled and priced")
	mustContain(t, page, `"@type":"Product"`, "structured data describes the product")
	mustContain(t, page, `-w480.png 480w`, "product pictures are responsive")
	mustContain(t, get(t, handler, "/sitemap.xml").Body.String(), "<loc>https://wildflower.example/shop/sourdough-loaf</loc>", "the sitemap lists the product")

	// The product block on a page.
	pageID := firstPageID(t, host, "menu")
	postJSON(t, handler, "/admin/api/pages/"+pageID, `{"title":"Menu","slug":"menu","blocks":[{"kind":"product","product":"product:`+id+`"}]}`)
	editor := get(t, handler, "/admin/edit/"+pageID).Body.String()
	mustContain(t, editor, `data-product-select="true"`, "the canvas offers the product picker")
	mustContain(t, editor, `data-products-presets="true"`, "and presets for new blocks")
	post(t, handler, "/admin/api/pages/"+pageID+"/publish", url.Values{})
	mustContain(t, get(t, handler, "/menu").Body.String(), `class="site-products site-products--single"`, "the product card renders on the page")

	// The cart: add, cap at stock, update, remove.
	rec = post(t, handler, "/cart/add", url.Values{"product": {id}, "variant": {"small"}, "qty": {"5"}})
	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "added=1") {
		t.Fatalf("add to cart = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	cookie := rec.Result().Cookies()
	if len(cookie) == 0 || cookie[0].Name != cartCookie || !cookie[0].HttpOnly {
		t.Fatalf("cart cookie = %+v", cookie)
	}
	cart := getWithCookie(t, handler, "/cart", cookie[0]).Body.String()
	mustContain(t, cart, "Sourdough loaf — Small", "the cart names the option")
	mustContain(t, cart, `name="qty_0" value="2"`, "quantity is capped at the 2 in stock")
	mustContain(t, cart, "adjusted a quantity", "and the visitor is told")
	mustContain(t, cart, "Subtotal", "there is a subtotal")
	mustContain(t, cart, "$13.00", "two loaves at 6.50")
	mustContain(t, cart, "t open yet.", "no checkout until payments are connected")

	rec = postWithCookie(t, handler, "/cart/update", url.Values{"qty_0": {"1"}}, cookie[0])
	cart = getWithCookie(t, handler, "/cart", rec.Result().Cookies()[0]).Body.String()
	mustContain(t, cart, "$6.50", "one loaf after the update")
	rec = postWithCookie(t, handler, "/cart/update", url.Values{"remove": {"0"}}, rec.Result().Cookies()[0])
	if c := rec.Result().Cookies()[0]; c.MaxAge != -1 {
		t.Fatalf("an empty cart clears the cookie: %+v", c)
	}
	mustContain(t, get(t, handler, "/cart").Body.String(), "Your cart is empty", "the empty cart says so")

	// A sold-out product cannot be added, and a page cannot take /shop.
	if rec := post(t, handler, "/cart/add", url.Values{"product": {id}, "variant": {"large"}, "qty": {"1"}}); strings.Contains(get(t, handler, "/cart").Body.String(), "Large") {
		t.Fatalf("sold-out option was added: %q", rec.Header().Get("Location"))
	}
	mustContain(t, post(t, handler, "/admin/pages", url.Values{"title": {"Shop"}, "slug": {"shop"}}).Body.String(), "reserved", "/shop is reserved for the shop")

	// Currency changes every price.
	postSettings(t, handler, map[string]string{"title": "Wildflower Bakery", "currency": "gbp", "countVisitors": "on"}, nil)
	mustContain(t, get(t, handler, "/shop").Body.String(), "£6.50", "prices follow the currency setting")
}

func getWithCookie(t *testing.T, handler http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func postWithCookie(t *testing.T, handler http.Handler, path string, form url.Values, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// postMultipart posts a multipart form with the CSRF token, the way the
// product editor's form submits.
func postMultipart(t *testing.T, handler http.Handler, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("_csrf", csrfToken(t, handler))
	for key, values := range form {
		for _, value := range values {
			_ = writer.WriteField(key, value)
		}
	}
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
