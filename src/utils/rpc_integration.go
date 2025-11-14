package utils

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// DopamintRPCClient wraps the standard RPC client with Dopamint-specific filtering
type DopamintRPCClient struct {
	client         *ethclient.Client
	addressFilter  []common.Address
	filterEnabled  bool
}

// NewDopamintRPCClient creates a new Dopamint RPC client
func NewDopamintRPCClient(rpcURL string, addresses []common.Address, filterEnabled bool) (*DopamintRPCClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RPC: %w", err)
	}

	return &DopamintRPCClient{
		client:        client,
		addressFilter: addresses,
		filterEnabled: filterEnabled,
	}, nil
}

// GetFilteredLogs fetches logs with address filtering
func (d *DopamintRPCClient) GetFilteredLogs(ctx context.Context, fromBlock, toBlock *big.Int) ([]types.Log, error) {
	if !d.filterEnabled || len(d.addressFilter) == 0 {
		// No filtering, fetch all logs
		query := ethereum.FilterQuery{
			FromBlock: fromBlock,
			ToBlock:   toBlock,
		}
		return d.client.FilterLogs(ctx, query)
	}

	// Fetch logs only from Dopamint contracts
	query := ethereum.FilterQuery{
		FromBlock: fromBlock,
		ToBlock:   toBlock,
		Addresses: d.addressFilter,
	}

	logs, err := d.client.FilterLogs(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch filtered logs: %w", err)
	}

	fmt.Printf("[DopamintRPC] Fetched %d logs from %d contracts (blocks %s-%s)\n",
		len(logs), len(d.addressFilter), fromBlock.String(), toBlock.String())

	return logs, nil
}

// UpdateAddressFilter updates the address filter
func (d *DopamintRPCClient) UpdateAddressFilter(addresses []common.Address) {
	d.addressFilter = addresses
	fmt.Printf("[DopamintRPC] Updated address filter: %d addresses\n", len(addresses))
}

// GetBlockNumber gets the latest block number
func (d *DopamintRPCClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	return d.client.BlockNumber(ctx)
}

// GetBlockByNumber gets a block by number
func (d *DopamintRPCClient) GetBlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error) {
	return d.client.BlockByNumber(ctx, number)
}

// Close closes the RPC connection
func (d *DopamintRPCClient) Close() {
	d.client.Close()
}

// GetClient returns the underlying eth client
func (d *DopamintRPCClient) GetClient() *ethclient.Client {
	return d.client
}

// LogFilterStats represents filtering statistics
type LogFilterStats struct {
	TotalLogsReceived  int64
	LogsAfterFilter    int64
	BlocksProcessed    int64
	ContractsWatched   int
	FilterEnabled      bool
}

// CalculateFilterEfficiency calculates the efficiency of filtering
func CalculateFilterEfficiency(stats LogFilterStats) float64 {
	if stats.TotalLogsReceived == 0 {
		return 0
	}
	return (1.0 - float64(stats.LogsAfterFilter)/float64(stats.TotalLogsReceived)) * 100
}
