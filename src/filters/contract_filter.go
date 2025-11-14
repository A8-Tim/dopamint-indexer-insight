package filters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ContractFilter manages which contracts to index
type ContractFilter struct {
	mu                sync.RWMutex
	factoryAddress    common.Address
	paymentAddress    common.Address
	nftContracts      map[common.Address]bool
	enabled           bool
	autoDiscovery     bool
	mongodbSyncEnabled bool
	mongodbSyncInterval time.Duration
}

// ContractConfig represents the contract configuration
type ContractConfig struct {
	Network   string `json:"network"`
	ChainID   int    `json:"chainId"`
	Contracts struct {
		Factory struct {
			Address     string   `json:"address"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Events      []string `json:"events"`
		} `json:"factory"`
		Payment struct {
			Address     string   `json:"address"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Events      []string `json:"events"`
		} `json:"payment"`
		NFTContracts []string `json:"nftContracts"`
	} `json:"contracts"`
	EventFilters struct {
		Enabled     bool   `json:"enabled"`
		FilterMode  string `json:"filterMode"`
		Description string `json:"description"`
	} `json:"eventFilters"`
	SyncSettings struct {
		MongoDBSync struct {
			Enabled         bool   `json:"enabled"`
			IntervalSeconds int    `json:"intervalSeconds"`
			Description     string `json:"description"`
		} `json:"mongodbSync"`
		AutoDiscovery struct {
			Enabled     bool   `json:"enabled"`
			Description string `json:"description"`
		} `json:"autoDiscovery"`
	} `json:"syncSettings"`
}

// NewContractFilter creates a new contract filter
func NewContractFilter(configPath string) (*ContractFilter, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ContractConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	filter := &ContractFilter{
		factoryAddress:      common.HexToAddress(config.Contracts.Factory.Address),
		paymentAddress:      common.HexToAddress(config.Contracts.Payment.Address),
		nftContracts:        make(map[common.Address]bool),
		enabled:             config.EventFilters.Enabled,
		autoDiscovery:       config.SyncSettings.AutoDiscovery.Enabled,
		mongodbSyncEnabled:  config.SyncSettings.MongoDBSync.Enabled,
		mongodbSyncInterval: time.Duration(config.SyncSettings.MongoDBSync.IntervalSeconds) * time.Second,
	}

	// Load initial NFT contracts
	for _, addr := range config.Contracts.NFTContracts {
		if addr != "" {
			filter.nftContracts[common.HexToAddress(addr)] = true
		}
	}

	return filter, nil
}

// IsEnabled returns whether filtering is enabled
func (cf *ContractFilter) IsEnabled() bool {
	cf.mu.RLock()
	defer cf.mu.RUnlock()
	return cf.enabled
}

// ShouldIndexLog determines if a log should be indexed
func (cf *ContractFilter) ShouldIndexLog(address common.Address) bool {
	if !cf.enabled {
		return true // Index everything if filtering is disabled
	}

	cf.mu.RLock()
	defer cf.mu.RUnlock()

	// Check if it's the factory contract
	if address == cf.factoryAddress {
		return true
	}

	// Check if it's the payment contract
	if address == cf.paymentAddress {
		return true
	}

	// Check if it's a known NFT contract
	if cf.nftContracts[address] {
		return true
	}

	return false
}

// AddNFTContract adds a new NFT contract to the watch list
func (cf *ContractFilter) AddNFTContract(address common.Address) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	if !cf.nftContracts[address] {
		cf.nftContracts[address] = true
		fmt.Printf("[ContractFilter] Added NFT contract: %s (total: %d)\n", address.Hex(), len(cf.nftContracts))
	}
}

// AddNFTContracts adds multiple NFT contracts
func (cf *ContractFilter) AddNFTContracts(addresses []common.Address) {
	cf.mu.Lock()
	defer cf.mu.Unlock()

	newCount := 0
	for _, addr := range addresses {
		if !cf.nftContracts[addr] {
			cf.nftContracts[addr] = true
			newCount++
		}
	}

	if newCount > 0 {
		fmt.Printf("[ContractFilter] Added %d new NFT contracts (total: %d)\n", newCount, len(cf.nftContracts))
	}
}

// GetWatchedAddresses returns all addresses being watched
func (cf *ContractFilter) GetWatchedAddresses() []common.Address {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	addresses := make([]common.Address, 0, len(cf.nftContracts)+2)
	addresses = append(addresses, cf.factoryAddress)
	addresses = append(addresses, cf.paymentAddress)

	for addr := range cf.nftContracts {
		addresses = append(addresses, addr)
	}

	return addresses
}

// GetAddressFilter returns the address filter for RPC calls
// Returns lowercase hex addresses without 0x prefix for use in eth_getLogs
func (cf *ContractFilter) GetAddressFilter() []string {
	addresses := cf.GetWatchedAddresses()
	filter := make([]string, len(addresses))

	for i, addr := range addresses {
		filter[i] = strings.ToLower(addr.Hex())
	}

	return filter
}

// Stats returns statistics about the filter
func (cf *ContractFilter) Stats() map[string]interface{} {
	cf.mu.RLock()
	defer cf.mu.RUnlock()

	return map[string]interface{}{
		"enabled":             cf.enabled,
		"factory_address":     cf.factoryAddress.Hex(),
		"payment_address":     cf.paymentAddress.Hex(),
		"nft_contracts_count": len(cf.nftContracts),
		"total_watched":       len(cf.nftContracts) + 2,
		"auto_discovery":      cf.autoDiscovery,
		"mongodb_sync":        cf.mongodbSyncEnabled,
	}
}

// StartMongoDBSync starts the MongoDB sync goroutine
func (cf *ContractFilter) StartMongoDBSync(ctx context.Context, mongoClient MongoDBClient) {
	if !cf.mongodbSyncEnabled {
		fmt.Println("[ContractFilter] MongoDB sync is disabled")
		return
	}

	fmt.Printf("[ContractFilter] Starting MongoDB sync (interval: %s)\n", cf.mongodbSyncInterval)

	ticker := time.NewTicker(cf.mongodbSyncInterval)
	defer ticker.Stop()

	// Initial sync
	if err := cf.syncFromMongoDB(ctx, mongoClient); err != nil {
		fmt.Printf("[ContractFilter] Initial MongoDB sync error: %v\n", err)
	}

	// Periodic sync
	for {
		select {
		case <-ctx.Done():
			fmt.Println("[ContractFilter] MongoDB sync stopped")
			return
		case <-ticker.C:
			if err := cf.syncFromMongoDB(ctx, mongoClient); err != nil {
				fmt.Printf("[ContractFilter] MongoDB sync error: %v\n", err)
			}
		}
	}
}

// syncFromMongoDB fetches NFT contract addresses from MongoDB
func (cf *ContractFilter) syncFromMongoDB(ctx context.Context, mongoClient MongoDBClient) error {
	addresses, err := mongoClient.GetNFTContractAddresses(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch NFT contracts from MongoDB: %w", err)
	}

	if len(addresses) > 0 {
		cf.AddNFTContracts(addresses)
		fmt.Printf("[ContractFilter] Synced %d NFT contracts from MongoDB\n", len(addresses))
	}

	return nil
}

// MongoDBClient interface for fetching contract addresses
type MongoDBClient interface {
	GetNFTContractAddresses(ctx context.Context) ([]common.Address, error)
}
