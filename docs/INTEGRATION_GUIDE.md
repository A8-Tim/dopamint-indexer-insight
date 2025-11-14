# Integration Guide: Applying Dopamint Filters to Thirdweb Insight

This guide explains how to integrate the Dopamint custom filtering logic into the thirdweb insight codebase.

## Overview

The integration involves:
1. Adding the Dopamint filter modules to the insight codebase
2. Modifying the RPC client to use address filtering
3. Implementing the event listener for auto-discovery
4. Connecting to MongoDB for contract synchronization

## Step-by-Step Integration

### Step 1: Copy Custom Modules

Copy the custom Dopamint modules into the insight codebase:

```bash
cd /home/user/insight

# Create Dopamint integration directory
mkdir -p internal/dopamint

# Copy filter modules
cp /home/user/dopamint-indexer-insight/src/filters/*.go internal/dopamint/
cp /home/user/dopamint-indexer-insight/src/database/mongodb_client.go internal/dopamint/
cp /home/user/dopamint-indexer-insight/src/utils/rpc_integration.go internal/dopamint/
```

### Step 2: Install Dependencies

Add required Go modules:

```bash
cd /home/user/insight

# Add MongoDB driver
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/bson

# Update dependencies
go mod tidy
```

### Step 3: Modify RPC Client

Edit `/home/user/insight/internal/rpc/rpc.go`:

```go
package rpc

import (
    // ... existing imports
    "github.com/thirdweb-dev/indexer/internal/dopamint"
    "github.com/ethereum/go-ethereum"
)

type RPC struct {
    // ... existing fields
    contractFilter *dopamint.ContractFilter
    eventListener  *dopamint.EventListener
}

// NewRPC creates a new RPC client with Dopamint filtering
func NewRPC(config RPCConfig, contractFilter *dopamint.ContractFilter) (*RPC, error) {
    // ... existing initialization

    rpc := &RPC{
        // ... existing fields
        contractFilter: contractFilter,
        eventListener:  dopamint.NewEventListener(contractFilter, config.FactoryAddress),
    }

    return rpc, nil
}

// GetLogs fetches logs with address filtering
func (r *RPC) GetLogs(ctx context.Context, fromBlock, toBlock *big.Int) ([]types.Log, error) {
    // Check if filtering is enabled
    if r.contractFilter != nil && r.contractFilter.IsEnabled() {
        // Use filtered query
        addresses := r.contractFilter.GetWatchedAddresses()

        query := ethereum.FilterQuery{
            FromBlock: fromBlock,
            ToBlock:   toBlock,
            Addresses: addresses,
        }

        logs, err := r.client.FilterLogs(ctx, query)
        if err != nil {
            return nil, fmt.Errorf("failed to fetch filtered logs: %w", err)
        }

        // Process logs for auto-discovery
        r.eventListener.ProcessLogs(logs)

        log.Printf("[RPC] Fetched %d logs from %d contracts (blocks %s-%s)",
            len(logs), len(addresses), fromBlock.String(), toBlock.String())

        return logs, nil
    }

    // Fallback to unfiltered query
    return r.getLogsUnfiltered(ctx, fromBlock, toBlock)
}

// getLogsUnfiltered is the original implementation
func (r *RPC) getLogsUnfiltered(ctx context.Context, fromBlock, toBlock *big.Int) ([]types.Log, error) {
    // ... original implementation
}
```

### Step 4: Update Backfill Command

Edit `/home/user/insight/cmd/backfill.go`:

```go
package cmd

import (
    // ... existing imports
    "github.com/thirdweb-dev/indexer/internal/dopamint"
)

func backfillCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "backfill",
        Short: "Backfill historical blockchain data",
        Run: func(cmd *cobra.Command, args []string) {
            // Load configuration
            cfg := loadConfig()

            // Initialize Dopamint contract filter
            contractFilter, err := dopamint.NewContractFilter(cfg.Dopamint.ContractsConfigPath)
            if err != nil {
                log.Fatalf("Failed to initialize contract filter: %v", err)
            }

            // Initialize MongoDB client
            mongoConfig := dopamint.MongoDBConfig{
                URI:            cfg.MongoDB.URI,
                Database:       cfg.MongoDB.Database,
                Collection:     cfg.MongoDB.Collection,
                ConnectTimeout: 10 * time.Second,
            }

            mongoClient, err := dopamint.NewDopamintMongoClient(mongoConfig)
            if err != nil {
                log.Fatalf("Failed to connect to MongoDB: %v", err)
            }
            defer mongoClient.Close(context.Background())

            // Initial sync from MongoDB
            ctx := context.Background()
            addresses, err := mongoClient.GetNFTContractAddresses(ctx)
            if err != nil {
                log.Printf("Warning: Failed to sync from MongoDB: %v", err)
            } else {
                contractFilter.AddNFTContracts(addresses)
                log.Printf("Loaded %d NFT contracts from MongoDB", len(addresses))
            }

            // Initialize RPC client with filter
            rpcClient, err := rpc.NewRPC(cfg.RPC, contractFilter)
            if err != nil {
                log.Fatalf("Failed to initialize RPC client: %v", err)
            }

            // Print filter stats
            stats := contractFilter.Stats()
            log.Printf("Contract filter initialized: %+v", stats)

            // Run backfill
            backfillService := backfill.NewBackfillService(rpcClient, cfg.Storage)
            if err := backfillService.Run(ctx); err != nil {
                log.Fatalf("Backfill failed: %v", err)
            }
        },
    }

    return cmd
}
```

### Step 5: Update Committer Command

Edit `/home/user/insight/cmd/committer.go`:

```go
package cmd

import (
    // ... existing imports
    "github.com/thirdweb-dev/indexer/internal/dopamint"
)

func committerCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "committer",
        Short: "Run the committer service for live indexing",
        Run: func(cmd *cobra.Command, args []string) {
            cfg := loadConfig()

            // Initialize Dopamint contract filter
            contractFilter, err := dopamint.NewContractFilter(cfg.Dopamint.ContractsConfigPath)
            if err != nil {
                log.Fatalf("Failed to initialize contract filter: %v", err)
            }

            // Initialize MongoDB client
            mongoConfig := dopamint.MongoDBConfig{
                URI:            cfg.MongoDB.URI,
                Database:       cfg.MongoDB.Database,
                Collection:     cfg.MongoDB.Collection,
                ConnectTimeout: 10 * time.Second,
            }

            mongoClient, err := dopamint.NewDopamintMongoClient(mongoConfig)
            if err != nil {
                log.Fatalf("Failed to connect to MongoDB: %v", err)
            }
            defer mongoClient.Close(context.Background())

            // Start MongoDB sync in background
            ctx, cancel := context.WithCancel(context.Background())
            defer cancel()

            go contractFilter.StartMongoDBSync(ctx, mongoClient)

            // Initial sync
            addresses, err := mongoClient.GetNFTContractAddresses(ctx)
            if err != nil {
                log.Printf("Warning: Failed initial MongoDB sync: %v", err)
            } else {
                contractFilter.AddNFTContracts(addresses)
                log.Printf("Loaded %d NFT contracts from MongoDB", len(addresses))
            }

            // Initialize RPC client with filter
            rpcClient, err := rpc.NewRPC(cfg.RPC, contractFilter)
            if err != nil {
                log.Fatalf("Failed to initialize RPC client: %v", err)
            }

            // Print filter stats
            stats := contractFilter.Stats()
            log.Printf("Contract filter initialized: %+v", stats)

            // Run committer
            committerService := committer.NewCommitterService(rpcClient, cfg.Storage)
            if err := committerService.Run(ctx); err != nil {
                log.Fatalf("Committer failed: %v", err)
            }
        },
    }

    return cmd
}
```

### Step 6: Update Configuration

Edit `/home/user/insight/configs/config.go` to add Dopamint config:

```go
package configs

type Config struct {
    // ... existing fields

    // Dopamint-specific configuration
    Dopamint DopamintConfig `yaml:"dopamint"`
    MongoDB  MongoDBConfig  `yaml:"mongodb"`
}

type DopamintConfig struct {
    ContractsConfigPath string          `yaml:"contractsConfigPath"`
    Contracts           ContractsConfig `yaml:"contracts"`
    Filtering           FilterConfig    `yaml:"filtering"`
}

type ContractsConfig struct {
    Factory FactoryConfig `yaml:"factory"`
    Payment PaymentConfig `yaml:"payment"`
}

type FactoryConfig struct {
    Address    string `yaml:"address"`
    StartBlock int64  `yaml:"startBlock"`
}

type PaymentConfig struct {
    Address    string `yaml:"address"`
    StartBlock int64  `yaml:"startBlock"`
}

type FilterConfig struct {
    Enabled       bool   `yaml:"enabled"`
    Mode          string `yaml:"mode"`
    AutoDiscovery bool   `yaml:"autoDiscovery"`
}

type MongoDBConfig struct {
    URI                 string `yaml:"uri"`
    Database            string `yaml:"database"`
    Collection          string `yaml:"collection"`
    SyncEnabled         bool   `yaml:"syncEnabled"`
    SyncIntervalSeconds int    `yaml:"syncIntervalSeconds"`
}
```

### Step 7: Build and Test

```bash
cd /home/user/insight

# Build
go build -o indexer .

# Test with filtering
./indexer backfill \
  --config /home/user/dopamint-indexer-insight/src/config/indexer_config.yaml

# Test committer
./indexer committer \
  --config /home/user/dopamint-indexer-insight/src/config/indexer_config.yaml
```

## Verification

### 1. Check Filter Initialization

Look for log messages:
```
Contract filter initialized: map[enabled:true factory_address:0x... nft_contracts_count:5 ...]
```

### 2. Verify Filtered Log Fetching

Look for:
```
[RPC] Fetched 150 logs from 7 contracts (blocks 1000-2000)
```

Instead of:
```
[RPC] Fetched 50000 logs (blocks 1000-2000)  # Unfiltered would be much higher
```

### 3. Check Auto-Discovery

When a new NFT contract is created:
```
[EventListener] Discovered new NFT contract: 0x...
[ContractFilter] Added NFT contract: 0x... (total: 6)
```

### 4. Verify MongoDB Sync

Every 5 minutes:
```
[ContractFilter] Synced 8 NFT contracts from MongoDB
```

## Troubleshooting

### Filter Not Working

**Symptom**: Indexing all logs, not just Dopamint contracts

**Solution**:
1. Check `dopamint.filtering.enabled: true` in config
2. Verify contract addresses are correct
3. Check RPC client has contractFilter set

```bash
# Add debug logging
log.Printf("[DEBUG] Filter enabled: %v", contractFilter.IsEnabled())
log.Printf("[DEBUG] Watched addresses: %v", contractFilter.GetWatchedAddresses())
```

### MongoDB Sync Not Working

**Symptom**: NFT contracts not being added

**Solution**:
1. Test MongoDB connection
```bash
mongosh $MONGODB_URI --eval "db.nft_contracts.find()"
```

2. Check collection name matches
3. Verify documents have `contractAddress` field

### Auto-Discovery Not Working

**Symptom**: New NFT contracts not automatically added

**Solution**:
1. Verify Factory address is correct
2. Check event signature matches:
```go
// Verify in logs
log.Printf("[DEBUG] NFTContractCreated signature: %v", dopamint.NFTContractCreatedSignature)
```

3. Ensure backfill has processed Factory creation events

## Performance Tuning

### Optimal Block Ranges

With filtering enabled:
```yaml
rpc:
  logs:
    blocksPerRequest: 100  # Can be higher with filtering
```

Without filtering:
```yaml
rpc:
  logs:
    blocksPerRequest: 10  # Must be lower to avoid rate limits
```

### Memory Usage

Filtering reduces memory by ~90%:
- **Unfiltered**: ~5GB for 100K blocks
- **Filtered**: ~500MB for 100K blocks

### RPC Request Reduction

Expected reduction:
- **Full chain**: 100,000 logs per 1000 blocks
- **Dopamint only**: ~100 logs per 1000 blocks
- **Savings**: 99.9%

## Next Steps

1. **Monitor Performance**: Track filtering efficiency in Grafana
2. **Scale Up**: Increase `blocksPerRequest` gradually
3. **Optimize Queries**: Add indexes to ClickHouse for common queries
4. **API Layer**: Build REST API on top of ClickHouse

## Support

For issues with integration:
1. Check logs for error messages
2. Verify all config values
3. Test each component independently
4. Open an issue with logs and config
