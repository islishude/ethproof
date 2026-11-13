package proof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gofrs/flock"
)

type finalityFileStore struct{ dir string }

const maxFinalityCacheBytes = 4 << 20

// NewFinalityFileStore returns an atomic, permission-restricted local file store.
func NewFinalityFileStore(directory string) FinalityStore { return &finalityFileStore{directory} }
func (s *finalityFileStore) Lock(ctx context.Context) (func() error, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return nil, err
	}
	lock := flock.New(filepath.Join(s.dir, "state.lock"))
	ok, err := lock.TryLockContext(ctx, 50*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ctx.Err()
	}
	return lock.Unlock, nil
}
func (s *finalityFileStore) Load(ctx context.Context) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(filepath.Join(s.dir, "state.json"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, maxFinalityCacheBytes+1))
	if len(b) > maxFinalityCacheBytes {
		return nil, fmt.Errorf("%w: file too large", ErrFinalityCache)
	}
	return b, err
}
func (s *finalityFileStore) Save(ctx context.Context, b []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// Never replace trusted state with a file that Load would reject. Beacon
	// execution JSON can carry ignored fields larger than the cache limit.
	if len(b) > maxFinalityCacheBytes {
		return fmt.Errorf("%w: file too large", ErrFinalityCache)
	}
	f, err := os.CreateTemp(s.dir, ".state-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err = f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(s.dir, "state.json"))
}

type finalityCache struct {
	Version      int                `json:"version"`
	Fingerprint  common.Hash        `json:"fingerprint"`
	State        finalityState      `json:"state"`
	InitialTrust FinalityTrustEvent `json:"initialTrust"`
	LastTrust    FinalityTrustEvent `json:"lastTrust"`
}

func (c *FinalityClient) load(ctx context.Context) (*finalityCache, error) {
	b, err := c.store.Load(ctx)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cache finalityCache
	if err = json.Unmarshal(b, &cache); err != nil {
		return nil, fmt.Errorf("%w: malformed JSON: %w", ErrFinalityCache, err)
	}
	if cache.Version != 1 || cache.Fingerprint != c.fingerprint || cache.State.Current == nil || cache.LastTrust.Root == (common.Hash{}) {
		return nil, fmt.Errorf("%w: version, configuration or state mismatch", ErrFinalityCache)
	}
	if _, err = verifyFinalityHeader(cache.State.Finalized, c.network); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrFinalityCache, err)
	}
	if _, err = cache.State.Current.Deserialize(); err != nil {
		return nil, fmt.Errorf("%w: committee", ErrFinalityCache)
	}
	if cache.State.Next != nil {
		if _, err = cache.State.Next.Deserialize(); err != nil {
			return nil, fmt.Errorf("%w: next committee", ErrFinalityCache)
		}
	}
	return &cache, nil
}
func (c *FinalityClient) save(ctx context.Context, cache *finalityCache) error {
	b, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return c.store.Save(ctx, b)
}
