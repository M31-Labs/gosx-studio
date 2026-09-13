package sitehost

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"m31labs.dev/gosx"
)

// goods.go is what a product can be beyond a thing in a box: a file to
// download, or a subscription that renews.
//
// A digital product carries one file, stored beside the site data under a
// content hash. Paying for it earns a signed link that works for a week,
// shown on the thank-you page, in the receipt email, and on the order. A
// subscription is a product whose price repeats monthly or yearly; Stripe
// Checkout runs in subscription mode and the site keeps the subscription's
// status from the webhooks Stripe sends about it.

const (
	kindPhysical     = "physical"
	kindDigital      = "digital"
	kindSubscription = "subscription"
	kindBooking      = "booking"
	downloadPrefix   = "/download/"
	downloadTTL      = 7 * 24 * time.Hour
	fileMaxBytes     = 100 << 20
)

var fileExt = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)

func normalizeKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case kindDigital:
		return kindDigital
	case kindSubscription:
		return kindSubscription
	case kindBooking:
		return kindBooking
	default:
		return kindPhysical
	}
}

func kindLabel(kind string) string {
	switch normalizeKind(kind) {
	case kindDigital:
		return "Digital download"
	case kindSubscription:
		return "Subscription"
	case kindBooking:
		return "Bookable service"
	default:
		return "Physical item"
	}
}

func normalizeInterval(interval string) string {
	switch strings.ToLower(strings.TrimSpace(interval)) {
	case "year", "yearly", "annual":
		return "year"
	case "week", "weekly":
		return "week"
	default:
		return "month"
	}
}

// priceLabel is the price as the shop shows it: "$9.00 / month" for a
// subscription, plain for everything else.
func priceLabel(product Product, currency string) string {
	label := formatMoney(product.Price, currency)
	if product.Kind == kindSubscription {
		label += " / " + normalizeInterval(product.Interval)
	}
	return label
}

func (o Options) filesDir() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "files")
}

// storeFile keeps an uploaded file under its content hash, so the same file
// twice is stored once and a name can never point outside the folder.
func (h *Host) storeFile(r io.Reader, original string) (string, error) {
	dir := h.opts.filesDir()
	if dir == "" {
		return "", errors.New("files are not configured")
	}
	data, err := io.ReadAll(io.LimitReader(r, fileMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(data) > fileMaxBytes {
		return "", errors.New("too large")
	}
	if len(data) == 0 {
		return "", errors.New("empty")
	}
	ext := strings.ToLower(filepath.Ext(original))
	if !fileExt.MatchString(ext) {
		ext = ".bin"
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:12]) + ext
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	target := filepath.Join(dir, name)
	if _, err := os.Stat(target); err == nil {
		return name, nil
	}
	temp, err := os.CreateTemp(dir, "file-*")
	if err != nil {
		return "", err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		os.Remove(temp.Name())
		return "", err
	}
	if err := temp.Close(); err != nil {
		os.Remove(temp.Name())
		return "", err
	}
	if err := os.Rename(temp.Name(), target); err != nil {
		os.Remove(temp.Name())
		return "", err
	}
	return name, nil
}

var storedFileName = regexp.MustCompile(`^[a-f0-9]{24}\.[a-z0-9]{1,8}$`)

// ---------- download links ----------

type downloadLink struct {
	Name string
	URL  string
}

// downloadToken signs "order:file:expiry" so a link proves the purchase
// without a database lookup on the way in.
func (h *Host) downloadToken(orderID, stored string) string {
	expiry := strconv.FormatInt(timeNow().Add(downloadTTL).Unix(), 10)
	value := orderID + ":" + stored + ":" + expiry
	return base64.RawURLEncoding.EncodeToString([]byte(value + ":" + h.sign(value)))
}

func (h *Host) parseDownloadToken(token string) (orderID, stored string, ok bool) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", "", false
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 4 || !tokensEqual(parts[3], h.sign(strings.Join(parts[:3], ":"))) {
		return "", "", false
	}
	expiry, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || timeNow().Unix() > expiry {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// downloadsFor lists the files an order has paid for.
func (h *Host) downloadsFor(order Order, base string) []downloadLink {
	if order.Status != "paid" && order.Status != "fulfilled" {
		return nil
	}
	links := []downloadLink{}
	for _, line := range order.Lines {
		if line.File == "" {
			continue
		}
		links = append(links, downloadLink{Name: firstNonEmpty(line.FileName, line.Name), URL: base + downloadPrefix + h.downloadToken(order.ID, line.File)})
	}
	return links
}

func (h *Host) mountGoods(mux *http.ServeMux) {
	mux.HandleFunc("GET "+downloadPrefix+"{token}", h.handleDownload)
}

func (h *Host) handleDownload(w http.ResponseWriter, r *http.Request) {
	orderID, stored, ok := h.parseDownloadToken(r.PathValue("token"))
	if !ok || !storedFileName.MatchString(stored) {
		h.servePublicNotFound(w, h.settings(), "download")
		return
	}
	order, found := h.orders.get(orderID)
	if !found || (order.Status != "paid" && order.Status != "fulfilled") {
		h.servePublicNotFound(w, h.settings(), "download")
		return
	}
	name := ""
	for _, line := range order.Lines {
		if line.File == stored {
			name = firstNonEmpty(line.FileName, line.Name+filepath.Ext(stored))
		}
	}
	if name == "" {
		h.servePublicNotFound(w, h.settings(), "download")
		return
	}
	dir := h.opts.filesDir()
	if dir == "" {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(filepath.Join(dir, stored))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(name, `"`, "")+`"`)
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, name, info.ModTime(), file)
}

func renderDownloads(links []downloadLink) gosx.Node {
	if len(links) == 0 {
		return gosx.Fragment()
	}
	items := make([]gosx.Node, 0, len(links))
	for _, link := range links {
		items = append(items, gosx.El("li", nil, gosx.El("a", gosx.Attrs(gosx.Attr("href", link.URL), gosx.Attr("class", "site-download")), gosx.Text("Download "+link.Name))))
	}
	return gosx.El("div", gosx.Attrs(gosx.Attr("class", "site-downloads")),
		gosx.El("h3", nil, gosx.Text("Your downloads")),
		gosx.El("ul", gosx.Attrs(gosx.Attr("class", "site-list")), gosx.Fragment(items...)),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "site-post-meta")), gosx.Text("These links work for seven days. They're also in your receipt email.")),
	)
}

// newReceiptMail is the buyer's copy of the order, with any downloads.
func (h *Host) newReceiptMail(order Order, base string) Mail {
	siteTitle := firstNonEmpty(h.settings().Title, h.opts.SiteTitle)
	var b strings.Builder
	b.WriteString("Thanks for your order " + order.label() + " from " + siteTitle + ".\n\n")
	for _, line := range order.Lines {
		b.WriteString(strconv.Itoa(line.Qty) + " × " + line.Name + " — " + formatMoney(line.Total, order.Currency) + "\n")
	}
	b.WriteString("Total paid: " + formatMoney(order.Total, order.Currency) + "\n")
	if links := h.downloadsFor(order, base); len(links) > 0 {
		b.WriteString("\nYour downloads (links work for seven days):\n")
		for _, link := range links {
			b.WriteString(link.Name + ": " + link.URL + "\n")
		}
	}
	if order.StripeSubscription != "" {
		b.WriteString("\nThis is a subscription; it renews automatically. Reply to this email if you'd like to change or cancel it.\n")
	}
	if order.Ships {
		b.WriteString("\nWe'll send it to: " + order.ShipTo.oneLine() + "\n")
	}
	b.WriteString("\nQuestions? Just reply to this email.\n")
	return Mail{To: order.CustomerEmail, Subject: "Your order " + order.label() + " from " + siteTitle, Text: b.String(), ReplyTo: h.notifyAddress()}
}
