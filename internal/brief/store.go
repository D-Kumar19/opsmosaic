package brief

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Store struct {
	ConfigPath string
	CachePath  string
}

func DefaultStore() (Store, error) {
	if home := os.Getenv("OPSMOSAIC_HOME"); home != "" {
		return Store{filepath.Join(home, "config.json"), filepath.Join(home, "cache.json")}, nil
	}
	config, err := os.UserConfigDir()
	if err != nil {
		return Store{}, fmt.Errorf("locate configuration directory: %w", err)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return Store{}, fmt.Errorf("locate cache directory: %w", err)
	}
	return Store{filepath.Join(config, "opsmosaic", "config.json"), filepath.Join(cache, "opsmosaic", "cache.json")}, nil
}

func readJSON(path string, value interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	d := json.NewDecoder(io.LimitReader(f, 16<<20))
	if err := d.Decode(value); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var extra interface{}
	if err := d.Decode(&extra); err != io.EOF {
		return fmt.Errorf("read %s: unexpected trailing data", path)
	}
	return nil
}

func writeJSON(path string, value interface{}) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".opsmosaic-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary state: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(f.Name()); removeErr != nil && !os.IsNotExist(removeErr) && err == nil {
			err = fmt.Errorf("remove temporary state: %w", removeErr)
		}
	}()
	d := json.NewEncoder(f)
	d.SetIndent("", "  ")
	if encodeErr := d.Encode(value); encodeErr != nil {
		f.Close()
		return fmt.Errorf("encode state: %w", encodeErr)
	}
	if syncErr := f.Sync(); syncErr != nil {
		f.Close()
		return fmt.Errorf("sync state: %w", syncErr)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close state: %w", err)
	}
	if err := os.Rename(f.Name(), path); err != nil {
		return fmt.Errorf("save %s: %w", path, err)
	}
	return nil
}

func (s Store) profile() (Profile, error) {
	var p Profile
	if err := readJSON(s.ConfigPath, &p); err != nil {
		return p, err
	}
	return p, p.Validate()
}

func (s Store) cache() (Cache, error) {
	c := Cache{Version: 1, Sources: map[string]SourceCache{}}
	if err := readJSON(s.CachePath, &c); err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if c.Version != 1 || c.Sources == nil {
		return c, fmt.Errorf("invalid or unsupported cache; move %s aside to rebuild", s.CachePath)
	}
	return c, nil
}
