package proof

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	bt "github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
)

type finalityHTTP struct {
	base   *url.URL
	client *http.Client
	name   string
}

// NewBeaconFinalitySource constructs a bounded, context-aware Beacon API transport.
func NewBeaconFinalitySource(endpoint string, client *http.Client) (BeaconFinalitySource, error) {
	return newFinalityHTTP(endpoint, "beacon", client)
}

// NewCheckpointSource constructs a trusted-checkpoint transport. The caller
// supplies an opaque name identifying the independently operated source.
func NewCheckpointSource(name, endpoint string, client *http.Client) (CheckpointSource, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("checkpoint source name is empty")
	}
	return newFinalityHTTP(endpoint, name, client)
}

func newFinalityHTTP(endpoint, name string, client *http.Client) (*finalityHTTP, error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.Fragment != "" {
		return nil, errors.New("invalid Beacon API URL")
	}
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &finalityHTTP{base: u, client: client, name: name}, nil
}

// finalityTransportError deliberately omits URL/body/error text, which may
// contain credentials. Unwrap preserves cancellation and transport error identity.
type finalityTransportError struct{ cause error }

func (e *finalityTransportError) Error() string { return "beacon API request failed" }
func (e *finalityTransportError) Unwrap() error { return e.cause }

func (h *finalityHTTP) get(ctx context.Context, path string, query url.Values, out any) error {
	u := *h.base
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawPath = ""
	q := u.Query()
	maps.Copy(q, query)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return &finalityTransportError{err}
	}
	req.Header.Set("Accept", "application/json")
	// Do not let redirects move API credentials to another endpoint.
	client := *h.client
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return &finalityTransportError{err}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("beacon API HTTP status %d", resp.StatusCode)
	}
	const limit = 16 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return &finalityTransportError{err}
	}
	if len(data) > limit {
		return errors.New("beacon API response exceeds 16 MiB")
	}
	if err := json.Unmarshal(data, out); err != nil {
		return errors.New("invalid Beacon API JSON response")
	}
	return nil
}

type finalityEnvelope[T any] struct {
	Version string `json:"version"`
	Data    *T     `json:"data"`
}

func (e finalityEnvelope[T]) check() error {
	if !supportedFinalityFork(e.Version) {
		return fmt.Errorf("%w: response version", ErrFinalityUnsupported)
	}
	if e.Data == nil {
		return errors.New("missing Beacon API data")
	}
	return nil
}

func (h *finalityHTTP) Bootstrap(ctx context.Context, root common.Hash) (FinalityBootstrap, error) {
	var response finalityEnvelope[FinalityBootstrap]
	err := h.get(ctx, "/eth/v1/beacon/light_client/bootstrap/"+root.Hex(), nil, &response)
	if err == nil {
		err = response.check()
	}
	if err != nil {
		return FinalityBootstrap{}, err
	}
	return *response.Data, nil
}
func (h *finalityHTTP) Updates(ctx context.Context, period, count uint64) ([]FinalityUpdate, error) {
	if count == 0 || count > 128 {
		return nil, errors.New("invalid light client update count")
	}
	var response []finalityEnvelope[FinalityUpdate]
	query := url.Values{"start_period": {fmt.Sprint(period)}, "count": {fmt.Sprint(count)}}
	if err := h.get(ctx, "/eth/v1/beacon/light_client/updates", query, &response); err != nil {
		return nil, err
	}
	if uint64(len(response)) > count {
		return nil, errors.New("too many light client updates")
	}
	out := make([]FinalityUpdate, len(response))
	for i, e := range response {
		if err := e.check(); err != nil {
			return nil, err
		}
		out[i] = *e.Data
	}
	return out, nil
}
func (h *finalityHTTP) FinalityUpdate(ctx context.Context) (FinalityUpdate, error) {
	var response finalityEnvelope[FinalityUpdate]
	err := h.get(ctx, "/eth/v1/beacon/light_client/finality_update", nil, &response)
	if err == nil {
		err = response.check()
	}
	if err != nil {
		return FinalityUpdate{}, err
	}
	return *response.Data, nil
}
func (h *finalityHTTP) Name() string { return h.name }
func (h *finalityHTTP) Genesis(ctx context.Context) (uint64, common.Hash, error) {
	var response struct {
		Data *struct {
			Time common.Decimal `json:"genesis_time"`
			Root common.Hash    `json:"genesis_validators_root"`
		} `json:"data"`
	}
	if err := h.get(ctx, "/eth/v1/beacon/genesis", nil, &response); err != nil {
		return 0, common.Hash{}, err
	}
	if response.Data == nil {
		return 0, common.Hash{}, errors.New("missing genesis")
	}
	return uint64(response.Data.Time), response.Data.Root, nil
}
func (h *finalityHTTP) FinalizedCheckpoint(ctx context.Context) (FinalityCheckpoint, error) {
	var response struct {
		Optimistic *bool `json:"execution_optimistic"`
		Data       *struct {
			Finalized FinalityCheckpoint `json:"finalized"`
		} `json:"data"`
	}
	if err := h.get(ctx, "/eth/v1/beacon/states/head/finality_checkpoints", nil, &response); err != nil {
		return FinalityCheckpoint{}, err
	}
	if response.Data == nil || response.Optimistic == nil || *response.Optimistic {
		return FinalityCheckpoint{}, errors.New("missing or optimistic checkpoint data")
	}
	return response.Data.Finalized, nil
}
func (h *finalityHTTP) CheckpointHeader(ctx context.Context, root common.Hash) (CheckpointHeader, error) {
	var response struct {
		Optimistic *bool `json:"execution_optimistic"`
		Finalized  *bool `json:"finalized"`
		Data       *struct {
			Root      common.Hash `json:"root"`
			Canonical *bool       `json:"canonical"`
			Header    *struct {
				Message bt.Header `json:"message"`
			} `json:"header"`
		} `json:"data"`
	}
	if err := h.get(ctx, "/eth/v1/beacon/headers/"+root.Hex(), nil, &response); err != nil {
		return CheckpointHeader{}, err
	}
	if response.Data == nil || response.Data.Header == nil || response.Data.Canonical == nil || response.Finalized == nil || response.Optimistic == nil {
		return CheckpointHeader{}, errors.New("missing checkpoint header fields")
	}
	if response.Data.Root != root || response.Data.Header.Message.Hash() != root {
		return CheckpointHeader{}, errors.New("checkpoint header root mismatch")
	}
	return CheckpointHeader{response.Data.Header.Message, *response.Data.Canonical, *response.Finalized, *response.Optimistic}, nil
}

func (c FinalityCheckpoint) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Epoch string      `json:"epoch"`
		Root  common.Hash `json:"root"`
	}{strconv.FormatUint(uint64(c.Epoch), 10), c.Root})
}
