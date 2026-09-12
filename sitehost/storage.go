package sitehost

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"m31labs.dev/gosx-studio/cms/store/file"
	"m31labs.dev/gosx-studio/cms/store/memory"
	"m31labs.dev/gosx-studio/cms/store/sqlite"
)

// storage.go picks the store behind a data path.
//
// A ".json" path is the original one-file snapshot, kept for tests and for
// anyone who likes a site they can read in an editor. Anything else is a
// SQLite database — the default from now on — and a site.json sitting
// beside a new database is imported on the first start, then renamed so it
// is never read twice.

const legacyJSONName = "site.json"

// openStore opens the store for path and reports the JSON file it migrated
// from, if any.
func openStore(path string) (LifecycleContentStore, string, error) {
	path = strings.TrimSpace(path)
	if strings.EqualFold(filepath.Ext(path), ".json") {
		store, err := file.Open(path)
		if err == nil {
			return store, "", nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, "", err
		}
		store, err = file.New(path, memory.Seed{})
		return store, "", err
	}

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		legacy := filepath.Join(filepath.Dir(path), legacyJSONName)
		if _, err := os.Stat(legacy); err == nil {
			old, err := file.Open(legacy)
			if err != nil {
				return nil, "", err
			}
			store, err := sqlite.New(path, old.Snapshot())
			if err != nil {
				return nil, "", err
			}
			if err := os.Rename(legacy, legacy+".migrated"); err != nil {
				store.Close()
				return nil, "", err
			}
			return store, legacy, nil
		}
	}
	store, err := sqlite.Open(path)
	if err != nil {
		return nil, "", err
	}
	return store, "", nil
}
