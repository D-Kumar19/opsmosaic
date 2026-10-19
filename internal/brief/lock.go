package brief

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var errStateBusy = errors.New("state is busy")

func (s Store) lock(ctx context.Context, wait time.Duration) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(s.ConfigPath), 0700); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	// Never unlink this file: all processes must lock the same stable inode.
	file, err := os.OpenFile(s.ConfigPath+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open state lock: %w", err)
	}
	deadline := time.NewTimer(wait)
	defer deadline.Stop()
	retry := time.NewTicker(10 * time.Millisecond)
	defer retry.Stop()
	for {
		acquired, err := tryLock(file)
		if err != nil {
			return nil, errors.Join(err, file.Close())
		}
		if acquired {
			return file, nil
		}
		select {
		case <-ctx.Done():
			return nil, errors.Join(ctx.Err(), file.Close())
		case <-deadline.C:
			return nil, errors.Join(errStateBusy, file.Close())
		case <-retry.C:
		}
	}
}
