package proof

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bp "github.com/ethereum/go-ethereum/beacon/params"
	bt "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

// FinalityClient performs on-demand, finalized-only light-client synchronization.
// It does not run background goroutines. Concurrent calls serialize through Store.
type FinalityClient struct {
	config        FinalityConfig
	network       *bp.ChainConfig
	chainID       *big.Int
	execution     HeaderSource
	beacon        BeaconFinalitySource
	checkpoints   []CheckpointSource
	store         FinalityStore
	now           func() time.Time
	onTrustChange func(FinalityTrustEvent)
	fingerprint   common.Hash
}

// NewFinalityClient uses the standard HTTP transport and local atomic file cache.
func NewFinalityClient(cfg FinalityConfig, execution HeaderSource) (*FinalityClient, error) {
	return NewFinalityClientWithDependencies(cfg, execution, FinalityDependencies{})
}

// ValidateFinalityConfig validates configuration without performing network or file I/O.
func ValidateFinalityConfig(cfg FinalityConfig) error {
	_, err := NewFinalityClient(cfg, &configOnlyHeaderSource{})
	return err
}

type configOnlyHeaderSource struct{ HeaderSource }

// Embedding the interface supplies its methods for configuration validation only.
// The object is never used for execution calls.

func NewFinalityClientWithDependencies(cfg FinalityConfig, execution HeaderSource, deps FinalityDependencies) (*FinalityClient, error) {
	if isNilSource(execution) {
		return nil, errors.New("execution source is nil")
	}
	var net *bp.ChainConfig
	var chain uint64
	switch cfg.Network {
	case "mainnet":
		net = bp.MainnetLightConfig
		chain = 1
	case "sepolia":
		net = bp.SepoliaLightConfig
		chain = 11155111
	default:
		return nil, errors.New("finality network must be mainnet or sepolia")
	}
	if cfg.MaxAncestry == 0 {
		cfg.MaxAncestry = 8192
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = 300
	}
	if cfg.TimeoutSeconds > 86400 {
		return nil, errors.New("finality timeout must not exceed one day")
	}
	if cfg.CacheDir == "" {
		dir, err := os.UserCacheDir()
		if err != nil {
			return nil, err
		}
		cfg.CacheDir = filepath.Join(dir, "ethproof", "finality", cfg.Network)
	}
	if isNilSource(deps.Beacon) {
		var err error
		deps.Beacon, err = NewBeaconFinalitySource(cfg.BeaconRPC, nil)
		if err != nil {
			return nil, err
		}
	}
	identities, err := configureCheckpointSources(cfg, &deps)
	if err != nil {
		return nil, err
	}
	if isNilSource(deps.Store) {
		deps.Store = NewFinalityFileStore(cfg.CacheDir)
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	sort.Strings(identities)
	// Bind persisted state to the exact trust policy and fork schedule. Endpoint
	// strings enter only this one-way digest; they are never persisted themselves.
	b, err := json.Marshal(struct {
		Network string
		Genesis common.Hash
		Time    uint64
		Forks   bp.Forks
		Sources []string
	}{cfg.Network, net.GenesisValidatorsRoot, net.GenesisTime, net.Forks, identities})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(b)
	return &FinalityClient{cfg, net, new(big.Int).SetUint64(chain), execution, deps.Beacon, deps.Checkpoints, deps.Store, deps.Now, deps.OnTrustChange, common.Hash(digest)}, nil
}

func configureCheckpointSources(cfg FinalityConfig, deps *FinalityDependencies) ([]string, error) {
	identities := make([]string, 0, len(cfg.CheckpointRPCs))
	if deps.Checkpoints == nil {
		seen := map[string]bool{}
		for i, endpoint := range cfg.CheckpointRPCs {
			h, err := newFinalityHTTP(endpoint, fmt.Sprintf("checkpoint[%d]", i), nil)
			if err != nil {
				return nil, err
			}
			// Distinct hosts prevent counting aliases with different credentials/paths as operators.
			host := strings.TrimSuffix(strings.ToLower(h.base.Hostname()), ".")
			if seen[host] {
				return nil, errors.New("checkpoint sources must have distinct hosts")
			}
			seen[host] = true
			deps.Checkpoints = append(deps.Checkpoints, h)
			identities = append(identities, h.base.String())
		}
	} else {
		for _, s := range deps.Checkpoints {
			if isNilSource(s) {
				return nil, errors.New("nil checkpoint source")
			}
			identities = append(identities, s.Name())
		}
	}
	if len(deps.Checkpoints) > 16 {
		return nil, errors.New("at most 16 checkpoint providers are supported")
	}
	if len(deps.Checkpoints) < 3 {
		return nil, errors.New("at least 3 checkpoint providers are required")
	}
	seenNames := map[string]bool{}
	for _, s := range deps.Checkpoints {
		if s.Name() == "" || seenNames[s.Name()] {
			return nil, errors.New("checkpoint source names must be nonempty and unique")
		}
		seenNames[s.Name()] = true
	}
	return identities, nil
}

func (c *FinalityClient) currentSlot() (uint64, error) {
	unix := c.now().Unix()
	if unix < 0 || uint64(unix) < c.network.GenesisTime {
		return 0, errors.New("clock precedes network genesis")
	}
	return (uint64(unix) - c.network.GenesisTime) / 12, nil
}
func freshFinalitySlot(slot, now uint64) bool { return slot <= now && now-slot < 7200 } // 24h, a startup policy, not a computed WSP.

// FinalizedAnchor authenticates every target block-context field. It uses the
// separately supplied execution source, never a source named in a proof package.
func (c *FinalityClient) FinalizedAnchor(ctx context.Context, target BlockContext) (anchor BlockAnchor, err error) {
	if c == nil {
		return anchor, errors.New("finality client is nil")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(c.config.TimeoutSeconds)*time.Second)
	defer cancel()
	if err = validateBlockContext(target); err != nil {
		return anchor, err
	}
	if target.ChainID.ToBig().Cmp(c.chainID) != 0 {
		return anchor, errors.New("proof and finality network chain IDs differ")
	}
	if err = verifyBlockContextAgainstSource(ctx, target, c.execution); err != nil {
		return anchor, err
	}
	unlock, err := c.store.Lock(ctx)
	if err != nil {
		return anchor, err
	}
	defer func() { err = errors.Join(err, unlock()) }()
	cache, err := c.load(ctx)
	if err != nil {
		return anchor, err
	}
	now, err := c.currentSlot()
	if err != nil {
		return anchor, err
	}
	if cache != nil && cache.State.Finalized.Beacon.Slot > now {
		return anchor, errors.New("cached finalized header is in the future")
	}
	if cache == nil || !freshFinalitySlot(cache.State.Finalized.Beacon.Slot, now) {
		cache, err = c.acquire(ctx, cache, now)
		if err != nil {
			return anchor, err
		}
		if err = c.save(ctx, cache); err != nil {
			return anchor, err
		}
		if c.onTrustChange != nil {
			c.onTrustChange(cache.LastTrust)
		}
	}
	// A cached finalized anchor is sufficient for its own ancestors. Sync only
	// when the target is newer; avoid walking long ancestry before checking its size.
	anchor, err = c.executionAnchor(ctx, cache.State.Finalized, target)
	if err == nil {
		return anchor, nil
	}
	if !errors.Is(err, ErrNotFinalized) {
		return anchor, err
	}
	if err = c.sync(ctx, cache, now); err != nil {
		return anchor, err
	}
	return c.executionAnchor(ctx, cache.State.Finalized, target)
}

func (c *FinalityClient) sync(ctx context.Context, cache *finalityCache, now uint64) error {
	period := cache.State.Finalized.Beacon.SyncPeriod()
	current := bt.SyncPeriod(now)
	if current < period {
		return errors.New("future cached period")
	}
	// Freshness bounds normal catchup to at most two periods. Keep explicit bounds
	// as defense in depth, including against a hostile injected source.
	if current-period > 128 {
		return ErrFinalityStale
	}
	updates, err := c.beacon.Updates(ctx, period, current-period+1)
	if err != nil {
		return err
	}
	if uint64(len(updates)) > current-period+1 {
		return errors.New("too many light client updates")
	}
	for _, update := range updates {
		if err = ctx.Err(); err != nil {
			return err
		}
		if !cache.State.updateRelevant(update) {
			continue
		}
		changed, e := cache.State.processUpdate(update, now, c.network)
		if e != nil {
			return e
		}
		if changed {
			if err = c.save(ctx, cache); err != nil {
				return err
			}
		}
	}
	update, err := c.beacon.FinalityUpdate(ctx)
	if err != nil {
		return err
	}
	// A latest finality update can repeat an already processed full update.
	if update.Attested.Beacon.Slot <= cache.State.Finalized.Beacon.Slot {
		return nil
	}
	changed, err := cache.State.processUpdate(update, now, c.network)
	if err != nil {
		return err
	}
	if changed {
		return c.save(ctx, cache)
	}
	return nil
}

func (c *FinalityClient) executionAnchor(ctx context.Context, finalized FinalityHeader, target BlockContext) (BlockAnchor, error) {
	hash, err := verifyFinalityHeader(finalized, c.network)
	if err != nil {
		return BlockAnchor{}, err
	}
	header, err := c.execution.HeaderByHash(ctx, hash)
	if err != nil {
		return BlockAnchor{}, err
	}
	block, err := BlockContextFromHeader(c.chainID, header)
	if err != nil {
		return BlockAnchor{}, err
	}
	if block.BlockHash != hash {
		return BlockAnchor{}, errors.New("finalized execution header hash mismatch")
	}
	if block.BlockNumber < target.BlockNumber {
		return BlockAnchor{}, ErrNotFinalized
	}
	distance := block.BlockNumber - target.BlockNumber
	if distance > c.config.MaxAncestry {
		return BlockAnchor{}, ErrFinalityHistory
	}
	for range distance {
		if err = ctx.Err(); err != nil {
			return BlockAnchor{}, err
		}
		parent, err := c.execution.HeaderByHash(ctx, block.ParentHash)
		if err != nil {
			return BlockAnchor{}, err
		}
		previous, err := BlockContextFromHeader(c.chainID, parent)
		if err != nil {
			return BlockAnchor{}, err
		}
		if previous.BlockHash != block.ParentHash || previous.BlockNumber != block.BlockNumber-1 {
			return BlockAnchor{}, errors.New("invalid execution ancestry")
		}
		block = previous
	}
	anchor := BlockAnchor(block)
	if err = verifyBlockContextAgainstAnchor(target, anchor); err != nil {
		return BlockAnchor{}, fmt.Errorf("%w: %w", ErrFinalityConflict, err)
	}
	return anchor, nil
}
