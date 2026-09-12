package sitehost

import (
	"archive/zip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"m31labs.dev/gosx"
)

// backup.go is "download everything" and the nightly copy nobody has to
// remember to make.
//
// A site here is a handful of files beside site.json: messages, forms,
// visitor counts, and the uploads folder. An export is those files in one
// zip, restorable by unzipping into a folder and starting the site from it.
// A backup is the same zip written into a backups folder once a day, with
// the last fourteen kept. Certificates stay out of both: they are private
// keys, and Let's Encrypt issues new ones for free.

const (
	backupKeep     = 14
	backupInterval = 24 * time.Hour
	backupCheck    = 30 * time.Minute
	exportRoot     = "site/"
)

var backupName = regexp.MustCompile(`^site-\d{4}-\d{2}-\d{2}-\d{4}\.zip$`)

func (o Options) backupDir() string {
	if o.DataPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(o.DataPath), "backups")
}

// siteFiles lists every file that belongs to the site, as (name in the
// archive, path on disk) pairs.
func (h *Host) siteFiles() [][2]string {
	dir := filepath.Dir(h.opts.DataPath)
	if h.opts.DataPath == "" {
		return nil
	}
	out := [][2]string{}
	for _, path := range []string{h.opts.DataPath, h.opts.messagesPath(), h.opts.formsPath(), h.opts.statsPath(), h.opts.productsPath(), h.opts.ordersPath(), h.opts.usersPath(), h.opts.auditPath()} {
		if path == "" {
			continue
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			out = append(out, [2]string{filepath.Base(path), path})
		}
	}
	if uploads := h.opts.uploadDir(); uploads != "" {
		entries, _ := os.ReadDir(uploads)
		for _, entry := range entries {
			if entry.IsDir() || !uploadName.MatchString(entry.Name()) && entry.Name() != "index.json" {
				continue
			}
			out = append(out, [2]string{"uploads/" + entry.Name(), filepath.Join(uploads, entry.Name())})
		}
	}
	_ = dir
	return out
}

func (h *Host) exportReadme() string {
	return `This is a complete copy of your website's data.

To restore it, or to move the site to another server:

  1. Unzip this file. You get a folder called "site".
  2. Start the site from it:  gosx-site -data site/` + filepath.Base(h.opts.DataPath) + ` -admin-password YOUR_PASSWORD
  3. Point your domain at the new server (Settings > Your own domain) if it moved.

Pictures are in site/uploads. Messages, forms, and visitor counts are the
JSON files. Certificates are not included: HTTPS gets new ones itself.
`
}

// writeArchive streams the site as a zip.
func (h *Host) writeArchive(w io.Writer) error {
	archive := zip.NewWriter(w)
	now := timeNow().UTC()
	add := func(name string, modTime time.Time, body io.Reader) error {
		header := &zip.FileHeader{Name: exportRoot + name, Method: zip.Deflate}
		header.Modified = modTime
		entry, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(entry, body)
		return err
	}
	if err := add("README.txt", now, strings.NewReader(h.exportReadme())); err != nil {
		return err
	}
	// A SQLite site is copied through the store, so the archive holds a
	// consistent database rather than a file caught mid-write.
	type backer interface{ Backup(io.Writer) error }
	for _, file := range h.siteFiles() {
		if file[1] == h.opts.DataPath {
			if store, ok := h.store.(backer); ok {
				header := &zip.FileHeader{Name: exportRoot + file[0], Method: zip.Deflate}
				header.Modified = now
				entry, err := archive.CreateHeader(header)
				if err != nil {
					return err
				}
				if err := store.Backup(entry); err != nil {
					return err
				}
				continue
			}
		}
		src, err := os.Open(file[1])
		if err != nil {
			continue
		}
		info, _ := src.Stat()
		modTime := now
		if info != nil {
			modTime = info.ModTime()
		}
		err = add(file[0], modTime, src)
		src.Close()
		if err != nil {
			return err
		}
	}
	return archive.Close()
}

// ---------- backups ----------

type backupInfo struct {
	Name string
	Size int64
	At   time.Time
}

func (h *Host) listBackups() []backupInfo {
	dir := h.opts.backupDir()
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]backupInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !backupName.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		out = append(out, backupInfo{Name: entry.Name(), Size: info.Size(), At: info.ModTime()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out
}

type backupState struct {
	mu        sync.Mutex
	lastCheck time.Time
	running   bool
}

// backupNow writes a backup and prunes old ones.
func (h *Host) backupNow() (backupInfo, error) {
	dir := h.opts.backupDir()
	if dir == "" {
		return backupInfo{}, errors.New("backups need a data folder")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return backupInfo{}, err
	}
	name := "site-" + timeNow().UTC().Format("2006-01-02-1504") + ".zip"
	temp, err := os.CreateTemp(dir, "backup-*")
	if err != nil {
		return backupInfo{}, err
	}
	if err := h.writeArchive(temp); err != nil {
		temp.Close()
		os.Remove(temp.Name())
		return backupInfo{}, err
	}
	if err := temp.Close(); err != nil {
		os.Remove(temp.Name())
		return backupInfo{}, err
	}
	target := filepath.Join(dir, name)
	if err := os.Rename(temp.Name(), target); err != nil {
		os.Remove(temp.Name())
		return backupInfo{}, err
	}
	// Keep the newest few; nothing else in the folder is touched.
	backups := h.listBackups()
	for index := backupKeep; index < len(backups); index++ {
		_ = os.Remove(filepath.Join(dir, backups[index].Name))
	}
	info, err := os.Stat(target)
	if err != nil {
		return backupInfo{Name: name}, nil
	}
	return backupInfo{Name: name, Size: info.Size(), At: info.ModTime()}, nil
}

// backupIfDue makes a backup when the newest one is a day old or missing.
// It is called from a request, so it is throttled and never runs twice at
// once.
func (h *Host) backupIfDue() {
	if h.opts.NoBackups || h.opts.backupDir() == "" || !h.SetupComplete() {
		return
	}
	now := timeNow()
	h.backups.mu.Lock()
	if h.backups.running || (!h.backups.lastCheck.IsZero() && now.Sub(h.backups.lastCheck) < backupCheck && now.After(h.backups.lastCheck)) {
		h.backups.mu.Unlock()
		return
	}
	h.backups.lastCheck = now
	h.backups.running = true
	h.backups.mu.Unlock()
	defer func() {
		h.backups.mu.Lock()
		h.backups.running = false
		h.backups.mu.Unlock()
	}()

	backups := h.listBackups()
	if len(backups) > 0 {
		newest, err := time.Parse("2006-01-02-1504", strings.TrimSuffix(strings.TrimPrefix(backups[0].Name, "site-"), ".zip"))
		if err == nil && now.UTC().Sub(newest) < backupInterval {
			return
		}
	}
	_, _ = h.backupNow()
}

// backupAsync runs the daily check in the background. Tests turn it off so
// a backup never outlives the request that started it.
var backupAsync = true

// housekeeping runs the daily backup check off the request path.
func (h *Host) housekeeping(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.opts.NoBackups && h.opts.backupDir() != "" {
			h.backups.mu.Lock()
			due := h.backups.lastCheck.IsZero() || timeNow().Sub(h.backups.lastCheck) >= backupCheck
			h.backups.mu.Unlock()
			if due && backupAsync {
				go h.backupIfDue()
			} else if due {
				h.backupIfDue()
			}
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- admin ----------

func (h *Host) mountBackups(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/export.zip", h.handleAdminExport)
	mux.HandleFunc("POST /admin/backups", h.handleAdminBackupNow)
	mux.HandleFunc("GET /admin/backups/{name}", h.handleAdminBackupDownload)
}

func (h *Host) handleAdminExport(w http.ResponseWriter, r *http.Request) {
	name := firstNonEmpty(normalizeSlug(h.settings().Title), "site") + "-export-" + timeNow().UTC().Format("2006-01-02") + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	_ = h.writeArchive(w)
}

func (h *Host) handleAdminBackupNow(w http.ResponseWriter, r *http.Request) {
	info, err := h.backupNow()
	if err != nil {
		http.Redirect(w, r, "/admin/settings?status="+queryEscape("We couldn't make a backup: "+err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/settings?status="+queryEscape("Backed up as "+info.Name+" ("+humanSize(info.Size)+")."), http.StatusSeeOther)
}

func (h *Host) handleAdminBackupDownload(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	dir := h.opts.backupDir()
	if dir == "" || !backupName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	file, err := os.Open(filepath.Join(dir, name))
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
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, name, info.ModTime(), file)
}

// renderBackupPanel is the Settings card: download everything, and the
// backups made for you.
func (h *Host) renderBackupPanel() gosx.Node {
	backups := h.listBackups()
	var list gosx.Node
	if h.opts.NoBackups {
		list = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("Automatic backups are turned off for this site."))
	} else if len(backups) == 0 {
		list = gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("A backup is made automatically once a day, and the last fourteen are kept. None yet."))
	} else {
		rows := make([]gosx.Node, 0, len(backups))
		for index, backup := range backups {
			if index == 7 {
				rows = append(rows, gosx.El("tr", nil, gosx.El("td", gosx.Attrs(gosx.Attr("colspan", "3")), gosx.Text("…and "+plural(len(backups)-7, "older backup")+" in the backups folder."))))
				break
			}
			rows = append(rows, gosx.El("tr", nil,
				gosx.El("td", nil, gosx.Text(formatWhen(backup.At))),
				gosx.El("td", nil, gosx.Text(humanSize(backup.Size))),
				gosx.El("td", gosx.Attrs(gosx.Attr("class", "admin-row-actions")),
					gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-row-btn"), gosx.Attr("href", "/admin/backups/"+backup.Name)), gosx.Text("Download"))),
			))
		}
		list = gosx.El("table", gosx.Attrs(gosx.Attr("class", "admin-table")),
			gosx.El("thead", nil, gosx.El("tr", nil, gosx.El("th", nil, gosx.Text("Made")), gosx.El("th", nil, gosx.Text("Size")), gosx.El("th", nil, gosx.Text("")))),
			gosx.El("tbody", nil, gosx.Fragment(rows...)))
	}
	return gosx.El("section", gosx.Attrs(gosx.Attr("class", "admin-panel")),
		gosx.El("h2", nil, gosx.Text("Backups and export")),
		gosx.El("p", nil, gosx.Text("Everything on this site — pages, posts, pictures, messages, forms, visitor counts — fits in one file. Keep a copy, or take it to another server.")),
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-actions")),
			gosx.El("a", gosx.Attrs(gosx.Attr("class", "admin-button"), gosx.Attr("href", "/admin/export.zip")), gosx.Text("Download everything")),
			gosx.El("form", gosx.Attrs(gosx.Attr("method", "post"), gosx.Attr("action", "/admin/backups"), gosx.Attr("class", "admin-inline-form")),
				h.csrfField(),
				gosx.El("button", gosx.Attrs(gosx.Attr("class", "admin-secondary"), gosx.Attr("type", "submit")), gosx.Text("Back up now")),
			),
		),
		gosx.El("h3", gosx.Attrs(gosx.Attr("class", "admin-subhead")), gosx.Text("Automatic backups")),
		list,
		gosx.El("p", gosx.Attrs(gosx.Attr("class", "admin-hint")), gosx.Text("To restore one: stop the site, unzip the backup, and start the site from the site.json inside it. The README in the zip has the exact command.")),
	)
}
