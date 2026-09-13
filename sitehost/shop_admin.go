package sitehost

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"m31labs.dev/gosx"
	cmsstore "m31labs.dev/gosx-studio/cms/store"
)

// shop_admin.go is the owner's side of the shop: the product list, the
// product editor, and the shop settings.

const checkoutPath = "/checkout"

// checkoutReady reports whether payments are connected. checkout.go owns the
// details; the cart only needs a yes or no.
func (h *Host) checkoutReady() bool {
	return strings.TrimSpace(h.settings().Metadata[stripeSecretKey]) != ""
}

const stripeSecretKey = "stripeSecretKey"

func (h *Host) handleAdminShop(w http.ResponseWriter, r *http.Request) {
	h.renderAdminShop(w, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderAdminShop(w http.ResponseWriter, status adminStatus) {
	products := h.products.list()
	currency := h.currency()

	var listing gosx.Node
	if len(products) == 0 {
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-empty")),
			gosx.El("h2", nil, gosx.Text("Nothing for sale yet")),
			gosx.El("p", nil, gosx.Text("Add a product below. Your shop appears at "+shopPath+" and in your menu as soon as one product is on sale, and any product can be dropped onto a page with the Product block.")),
		)
	} else {
		rows := make([]gosx.Node, 0, len(products))
		for _, product := range products {
			state, label := "draft", "Not on sale"
			switch {
			case product.Active && product.soldOut():
				state, label = "offline", "Sold out"
			case product.Active:
				state, label = "published", "On sale"
			}
			stock := "Not tracked"
			if product.TrackStock {
				total := product.Stock
				if len(product.Variants) > 0 {
					total = 0
					for _, variant := range product.Variants {
						total += variant.Stock
					}
				}
				stock = strconv.Itoa(total) + " in stock"
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/shop/"+product.ID)), gosx.Text(product.Name))),
				gosx.El("td", nil, gosx.Text(formatMoney(product.Price, currency))),
				gosx.El("td", nil, gosx.Text(stock)),
				gosx.El("td", nil, gosx.El("span", gosx.Attrs(gosx.Attr("class", "admin-badge"), gosx.Attr("data-state", state)), gosx.Text(label))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
					gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", "/admin/shop/"+product.ID)), gosx.Text("Edit")),
					gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", product.path()), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("View")),
				),
			))
		}
		listing = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Your products")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
				gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Product")), gosx.El("th", nil, gosx.Text("Price")), gosx.El("th", nil, gosx.Text("Stock")), gosx.El("th", nil, gosx.Text("Status")), gosx.El("th", nil, gosx.Text("")))),
				gosx.El("tbody", nil, gosx.Fragment(rows...)),
			),
		)
	}

	create := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Add a product")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/shop")),
			h.csrfField(),
			adminTextField("name", "Name", "", "What it's called on the shelf."),
			adminTextField("price", "Price", "", "In "+strings.ToUpper(currency)+", such as 12.50. Change the currency under Settings."),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit")), gosx.Text("Add product")),
			),
		),
	)
	var orders gosx.Node = gosx.Fragment()
	if h.checkoutReady() {
		orders = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/orders")), gosx.Text("Orders")))
	} else {
		orders = gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
			gosx.El("h2", nil, gosx.Text("Taking payments")),
			gosx.El("p", nil, gosx.Text("Visitors can fill a cart now. To let them pay, connect Stripe under Settings → Payments.")),
			gosx.El("p", nil, gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", "/admin/settings#payments")), gosx.Text("Connect payments"))),
		)
	}

	body := h.renderAdminShell("shop", h.shopTitle(),
		"What you sell, with pictures, options, and stock. Visitors buy from "+shopPath+".",
		status, listing, create, orders)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Shop"), body)
}

func (h *Host) handleAdminCreateProduct(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderAdminShop(w, adminStatus{Message: "We couldn't read that form. Try again.", Error: true})
		return
	}
	name := strings.TrimSpace(r.PostFormValue("name"))
	if name == "" {
		h.renderAdminShop(w, adminStatus{Message: "Give the product a name first.", Error: true})
		return
	}
	price, err := parseMoney(r.PostFormValue("price"), h.currency())
	if err != nil {
		h.renderAdminShop(w, adminStatus{Message: "That price doesn't look right. Try something like 12.50.", Error: true})
		return
	}
	product, err := h.products.put(Product{Name: name, Price: price, Ships: true})
	if err != nil {
		h.renderAdminShop(w, adminStatus{Message: "We couldn't add that product. Try again.", Error: true})
		return
	}
	http.Redirect(w, r, "/admin/shop/"+product.ID+"?status="+queryEscape("Added. Add a picture and a description, then put it on sale."), http.StatusSeeOther)
}

func (h *Host) handleAdminProduct(w http.ResponseWriter, r *http.Request) {
	product, ok := h.products.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "product")
		return
	}
	h.renderProductEditor(w, product, adminStatus{Message: r.URL.Query().Get("status")})
}

func (h *Host) renderProductEditor(w http.ResponseWriter, product Product, status adminStatus) {
	currency := h.currency()
	check := func(name string, on bool, label string) gosx.Node {
		attrs := []any{gosx.Attr("type", "checkbox"), gosx.Attr("name", name), gosx.Attr("value", "1")}
		if on {
			attrs = append(attrs, gosx.Attr("checked", "checked"))
		}
		return gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")), gosx.El("input", gosx.Attrs(attrs...)), gosx.Text(" "+label))
	}

	// Pictures: existing ones with alt text, plus room for one more by
	// address or by upload.
	pictures := make([]gosx.Node, 0, productMaxImages)
	for index, image := range product.Images {
		n := strconv.Itoa(index)
		var thumb gosx.Node = gosx.Fragment()
		if attrs, ok := h.imageAttrs(image.URL, "", "80px"); ok {
			thumb = gosx.El("img", gosx.Attrs(append(attrs, gosx.Attr("class", "admin-media__thumb admin-media__thumb--small"))...))
		}
		pictures = append(pictures, gosx.El("tr", nil,
			gosx.El("td", nil, thumb),
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "image_"+n), gosx.Attr("value", image.URL), gosx.Attr("aria-label", "Picture address")))),
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "alt_"+n), gosx.Attr("value", image.Alt), gosx.Attr("placeholder", "Describe it"), gosx.Attr("aria-label", "Picture description")))),
			gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("name", "removeImage"), gosx.Attr("value", n), gosx.Attr("aria-label", "Remove this picture")), gosx.Text("✕"))),
		))
	}
	var addPicture gosx.Node = gosx.Fragment()
	if len(product.Images) < productMaxImages {
		addPicture = gosx.Fragment(
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
				gosx.El("label", gosx.Attrs(gosx.Attr("for", "newImage")), gosx.Text("Add a picture")),
				gosx.El("input", gosx.Attrs(gosx.Attr("type", "file"), gosx.Attr("id", "newImage"), gosx.Attr("name", "newImage"), gosx.Attr("accept", "image/png,image/jpeg,image/gif,image/webp"))),
				gosx.El("small", nil, gosx.Text("Or paste an address from your Pictures library below.")),
			),
			adminTextField("newImageURL", "Picture address", "", "Something like /uploads/… from Pictures, or any https:// picture."),
		)
	}

	// Variants.
	variantRows := make([]gosx.Node, 0, len(product.Variants))
	for index, variant := range product.Variants {
		n := strconv.Itoa(index)
		variantRows = append(variantRows, gosx.El("tr", nil,
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "vname_"+n), gosx.Attr("value", variant.Name), gosx.Attr("aria-label", "Option name")))),
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "vprice_"+n), gosx.Attr("value", moneyInput(variant.Price, currency)), gosx.Attr("placeholder", "same"), gosx.Attr("aria-label", "Option price")))),
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "number"), gosx.Attr("name", "vstock_"+n), gosx.Attr("value", strconv.Itoa(variant.Stock)), gosx.Attr("min", "0"), gosx.Attr("aria-label", "Option stock")))),
			gosx.El("td", nil, gosx.El("input", gosx.Attrs(gosx.Attr("type", "text"), gosx.Attr("name", "vsku_"+n), gosx.Attr("value", variant.SKU), gosx.Attr("placeholder", "optional"), gosx.Attr("aria-label", "SKU")))),
			gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("type", "submit"), gosx.Attr("name", "removeVariant"), gosx.Attr("value", n), gosx.Attr("aria-label", "Remove this option")), gosx.Text("✕"))),
		))
	}
	var variantsTable gosx.Node = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("No options yet. Add one if this comes in sizes, colours, or bundles; each can have its own price and stock."))
	if len(product.Variants) > 0 {
		variantsTable = gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-fields")),
			gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Option")), gosx.El("th", nil, gosx.Text("Price")), gosx.El("th", nil, gosx.Text("Stock")), gosx.El("th", nil, gosx.Text("SKU")), gosx.El("th", nil, gosx.Text("")))),
			gosx.El("tbody", nil, gosx.Fragment(variantRows...)))
	}
	var stockField gosx.Node = gosx.Fragment()
	if len(product.Variants) == 0 {
		stockField = adminTextField("stock", "In stock", strconv.Itoa(product.Stock), "How many you have. Only used when stock is tracked.")
	}

	form := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/shop/"+product.ID), gosx.Attr("enctype", "multipart/form-data")),
			h.csrfField(),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "imageCount"), gosx.Attr("value", strconv.Itoa(len(product.Images))))),
			gosx.El("input", gosx.Attrs(gosx.Attr("type", "hidden"), gosx.Attr("name", "variantCount"), gosx.Attr("value", strconv.Itoa(len(product.Variants))))),
			gosx.El("h2", nil, gosx.Text("The basics")),
			adminTextField("name", "Name", product.Name, ""),
			adminTextField("slug", "Web address", product.Slug, "yoursite.com"+shopPath+"/…"),
			adminTextareaField("description", "Description", product.Description, "What it is, what it's made of, who it's for. **Bold** and _italic_ work."),
			adminTextField("price", "Price", moneyInput(product.Price, currency), "In "+strings.ToUpper(currency)+"."),
			adminTextField("compare", "Was price (optional)", moneyInput(product.Compare, currency), "Shown crossed out next to the price, for a sale."),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Pictures")),
			gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table admin-fields")), gosx.El("tbody", nil, gosx.Fragment(pictures...))),
			addPicture,
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Options")),
			variantsTable,
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit"), gosx.Attr("name", "addVariant"), gosx.Attr("value", "1")), gosx.Text("Add an option"))),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("What kind of product")),
			kindSelect(product),
			intervalSelect(product),
			h.digitalFileFields(product),
			renderBookingRuleFields(product),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Stock and shipping")),
			gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Only physical items track stock and ship.")),
			check("trackStock", product.TrackStock, "Track stock (the shop says \"sold out\" when it runs out)"),
			stockField,
			check("ships", product.Ships, "Needs shipping (a physical thing that gets posted)"),
			gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("On sale")),
			check("active", product.Active, "Show it in the shop"),
			gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-actions")),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("type", "submit"), gosx.Attr("name", "save"), gosx.Attr("value", "1")), gosx.Text("Save product")),
				gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("href", product.path()), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("View")),
			),
		),
	)
	remove := gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Remove")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Takes it out of the shop and off every page. Past orders keep their copy of it.")),
		gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/shop/"+product.ID+"/delete")),
			h.csrfField(),
			gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Remove this product")),
		),
	)
	body := h.renderAdminShell("shop", product.Name, "", status, form, remove)
	h.writeDocument(w, http.StatusOK, h.adminMeta("Product: "+product.Name), body)
}

func (h *Host) handleAdminProductAction(w http.ResponseWriter, r *http.Request) {
	product, ok := h.products.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "product")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		h.renderProductEditor(w, product, adminStatus{Message: "That was too big to read. Pictures up to 10 MB work best.", Error: true})
		return
	}
	currency := h.currency()
	fail := func(message string) {
		h.renderProductEditor(w, product, adminStatus{Message: message, Error: true})
	}

	product.Name = r.PostFormValue("name")
	product.Slug = r.PostFormValue("slug")
	product.Description = r.PostFormValue("description")
	price, err := parseMoney(r.PostFormValue("price"), currency)
	if err != nil {
		fail("That price doesn't look right. Try something like 12.50.")
		return
	}
	compare, err := parseMoney(r.PostFormValue("compare"), currency)
	if err != nil {
		fail("That \"was\" price doesn't look right.")
		return
	}
	product.Price, product.Compare = price, compare
	product.Kind = normalizeKind(r.PostFormValue("kind"))
	product.Interval = r.PostFormValue("interval")
	product.TrackStock = r.PostFormValue("trackStock") == "1"
	product.Ships = r.PostFormValue("ships") == "1"
	product.Active = r.PostFormValue("active") == "1"
	if r.PostFormValue("removeFile") == "1" {
		product.File, product.FileName = "", ""
	}
	if file, header, err := r.FormFile("newFile"); err == nil {
		stored, err := h.storeFile(file, header.Filename)
		file.Close()
		if err != nil {
			fail("That file didn't upload. Files up to 100 MB work.")
			return
		}
		product.File, product.FileName = stored, filepath.Base(header.Filename)
		if product.Kind == kindPhysical {
			product.Kind = kindDigital
		}
	}
	if product.Kind == kindBooking {
		product.Booking = bookingRulesFromForm(r)
	}
	if stock, err := strconv.Atoi(strings.TrimSpace(r.PostFormValue("stock"))); err == nil {
		product.Stock = stock
	}

	imageCount, _ := strconv.Atoi(r.PostFormValue("imageCount"))
	images := make([]ProductImage, 0, imageCount+1)
	for index := 0; index < imageCount && index < productMaxImages; index++ {
		n := strconv.Itoa(index)
		if r.PostFormValue("removeImage") == n {
			continue
		}
		images = append(images, ProductImage{URL: r.PostFormValue("image_" + n), Alt: r.PostFormValue("alt_" + n)})
	}
	if url := strings.TrimSpace(r.PostFormValue("newImageURL")); url != "" {
		images = append(images, ProductImage{URL: url})
	}
	if file, _, err := r.FormFile("newImage"); err == nil {
		url, err := h.storeUpload(file)
		file.Close()
		if err != nil {
			fail("That picture didn't upload. PNG, JPEG, GIF, and WebP up to 10 MB work.")
			return
		}
		images = append(images, ProductImage{URL: url})
	}
	product.Images = images

	variantCount, _ := strconv.Atoi(r.PostFormValue("variantCount"))
	variants := make([]Variant, 0, variantCount+1)
	for index := 0; index < variantCount && index < productMaxVars; index++ {
		n := strconv.Itoa(index)
		if r.PostFormValue("removeVariant") == n {
			continue
		}
		vprice, err := parseMoney(r.PostFormValue("vprice_"+n), currency)
		if err != nil {
			fail("An option's price doesn't look right.")
			return
		}
		stock, _ := strconv.Atoi(strings.TrimSpace(r.PostFormValue("vstock_" + n)))
		variants = append(variants, Variant{Name: r.PostFormValue("vname_" + n), Price: vprice, Stock: stock, SKU: r.PostFormValue("vsku_" + n)})
	}
	message := "Saved."
	if r.PostFormValue("addVariant") == "1" {
		variants = append(variants, Variant{Name: "New option"})
		message = "Added an option. Name it, and give it a price and stock if they differ."
	}
	product.Variants = variants

	saved, err := h.products.put(product)
	if err != nil {
		fail("We couldn't save that. Try again.")
		return
	}
	if saved.Active && len(saved.Images) == 0 {
		message += " Tip: products with a picture sell better."
	}
	http.Redirect(w, r, "/admin/shop/"+saved.ID+"?status="+queryEscape(message), http.StatusSeeOther)
}

func (h *Host) handleAdminProductDelete(w http.ResponseWriter, r *http.Request) {
	product, ok := h.products.get(r.PathValue("id"))
	if !ok {
		h.writeAdminNotFound(w, "product")
		return
	}
	if err := h.products.remove(product.ID); err != nil {
		http.Redirect(w, r, "/admin/shop?status="+queryEscape("We couldn't remove that. Try again."), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/shop?status="+queryEscape("Removed “"+product.Name+"”."), http.StatusSeeOther)
}

// ---------- shop settings ----------

func renderShopFields(settings cmsstore.SiteSettings) gosx.Node {
	current := normalizeCurrency(settings.Metadata[currencyKey])
	options := make([]gosx.Node, 0, len(Currencies()))
	for _, code := range Currencies() {
		attrs := []any{gosx.Attr("value", code)}
		if code == current {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(strings.ToUpper(code))))
	}
	return gosx.Fragment(
		gosx.El("h2", gosx.Attrs(gosx.Attr("class", "admin-subhead"), gosx.Attr("id", "shop")), gosx.Text("Shop")),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
			gosx.El("label", gosx.Attrs(gosx.Attr("for", "currency")), gosx.Text("Currency")),
			gosx.El("select", gosx.Attrs(gosx.Attr("id", "currency"), gosx.Attr("name", "currency")), gosx.Fragment(options...)),
			gosx.El("small", nil, gosx.Text("Every price on the site is in this currency.")),
		),
		adminTextField("shopTitle", "Shop name in the menu", settings.Metadata[shopTitleKey], "Leave blank for \"Shop\"."),
	)
}

func applyShopFields(r *http.Request, metadata cmsstore.Metadata) {
	metadata[currencyKey] = normalizeCurrency(r.PostFormValue("currency"))
	if title := strings.TrimSpace(r.PostFormValue("shopTitle")); title != "" {
		metadata[shopTitleKey] = title
	} else {
		delete(metadata, shopTitleKey)
	}
}

// renderProductPreview is a product on the canvas: the card with a picker to
// swap in another product.
func (h *Host) renderProductPreview(ref string) gosx.Node {
	products := h.products.list()
	if len(products) == 0 {
		return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-image-empty"), gosx.Attr("contenteditable", "false")),
			gosx.Text("No products yet. Add one under Shop, then pick it here."))
	}
	chosen, ok := h.productByRef(ref)
	if !ok {
		chosen = products[0]
	}
	options := make([]gosx.Node, 0, len(products))
	for _, product := range products {
		attrs := []any{gosx.Attr("value", product.ref())}
		if product.ID == chosen.ID {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(product.Name)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-product"), gosx.Attr("contenteditable", "false"), gosx.Attr("data-picker-target", "product")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-products site-products--single"), gosx.Attr("data-product-card", "true")), h.renderProductCard(chosen, true)),
		gosx.El("div", gosx.Attrs(gosx.Attr("class", "ed-form__bar")),
			gosx.El("label", nil, gosx.Text("Which product: "),
				gosx.El("select", gosx.Attrs(gosx.Attr("class", "ed-inline-select"), gosx.Attr("data-product-select", "true"), gosx.Attr("aria-label", "Which product")), gosx.Fragment(options...))),
			gosx.El("a", gosx.Attrs(gosx.Attr("href", "/admin/shop/"+chosen.ID), gosx.Attr("data-product-edit", "true"), gosx.Attr("target", "_blank"), gosx.Attr("rel", "noopener")), gosx.Text("Edit the product")),
		),
	)
}

type productPreset struct {
	Ref   string `json:"ref"`
	Name  string `json:"name"`
	Price string `json:"price"`
	Image string `json:"image"`
	Href  string `json:"href"`
	Edit  string `json:"edit"`
}

func (h *Host) productPresetsJSON() string {
	currency := h.currency()
	presets := make([]productPreset, 0, 8)
	for _, product := range h.products.list() {
		image := ""
		if len(product.Images) > 0 {
			image = product.Images[0].URL
		}
		presets = append(presets, productPreset{Ref: product.ref(), Name: product.Name, Price: formatMoney(product.Price, currency), Image: image, Href: product.path(), Edit: "/admin/shop/" + product.ID})
	}
	data, err := json.Marshal(presets)
	if err != nil {
		return "[]"
	}
	return strings.ReplaceAll(string(data), "</", "<\\/")
}

func kindSelect(product Product) gosx.Node {
	options := make([]gosx.Node, 0, 4)
	for _, kind := range []string{kindPhysical, kindDigital, kindSubscription, kindBooking} {
		attrs := []any{gosx.Attr("value", kind)}
		if normalizeKind(product.Kind) == kind {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text(kindLabel(kind))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "kind")), gosx.Text("This is a")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", "kind"), gosx.Attr("name", "kind")), gosx.Fragment(options...)),
		gosx.El("small", nil, gosx.Text("A physical item gets posted. A digital download is a file buyers get a link to. A subscription charges again every month or year. A bookable service has a calendar.")),
	)
}

func intervalSelect(product Product) gosx.Node {
	options := make([]gosx.Node, 0, 3)
	for _, interval := range []string{"month", "year", "week"} {
		attrs := []any{gosx.Attr("value", interval)}
		if normalizeInterval(product.Interval) == interval {
			attrs = append(attrs, gosx.Attr("selected", "selected"))
		}
		options = append(options, gosx.El("option", gosx.Attrs(attrs...), gosx.Text("every "+interval)))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "interval")), gosx.Text("Subscriptions charge")),
		gosx.El("select", gosx.Attrs(gosx.Attr("id", "interval"), gosx.Attr("name", "interval")), gosx.Fragment(options...)),
		gosx.El("small", nil, gosx.Text("Only used when this is a subscription. The price above is the amount each time.")),
	)
}

func (h *Host) digitalFileFields(product Product) gosx.Node {
	var current gosx.Node = gosx.El("small", nil, gosx.Text("No file yet. Buyers get a private link that works for seven days after paying."))
	if product.File != "" {
		current = gosx.Fragment(
			gosx.El("p", nil, gosx.Text("Current file: "), gosx.El("code", nil, gosx.Text(product.FileName))),
			gosx.El("label", gosx.Attrs(gosx.Attr("class", "admin-check")), gosx.El("input", gosx.Attrs(gosx.Attr("type", "checkbox"), gosx.Attr("name", "removeFile"), gosx.Attr("value", "1"))), gosx.Text(" Remove this file")),
		)
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "admin-field")),
		gosx.El("label", gosx.Attrs(gosx.Attr("for", "newFile")), gosx.Text("The file to download (digital products)")),
		gosx.El("input", gosx.Attrs(gosx.Attr("type", "file"), gosx.Attr("id", "newFile"), gosx.Attr("name", "newFile"))),
		current,
	)
}
