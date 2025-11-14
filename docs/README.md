# Dopamint Indexer - Custom Implementation Guide

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Key Features](#key-features)
4. [Prerequisites](#prerequisites)
5. [Installation](#installation)
6. [Configuration](#configuration)
7. [Running the Indexer](#running-the-indexer)
8. [Custom Filtering Implementation](#custom-filtering-implementation)
9. [MongoDB Integration](#mongodb-integration)
10. [Monitoring & Troubleshooting](#monitoring--troubleshooting)
11. [API Usage](#api-usage)
12. [Development Guide](#development-guide)

---

## Overview

The Dopamint Indexer is a customized version of [thirdweb-dev/insight](https://github.com/thirdweb-dev/insight) specifically designed to index only Dopamint project contracts on the Base chain.

### Why Custom Indexing?

Instead of indexing the entire Base blockchain (which would be massive), this indexer:
- **Filters by contract addresses**: Only indexes events from Dopamint Factory, Payment, and NFT contracts
- **Auto-discovers NFT contracts**: Automatically adds new NFT contracts when they're created by the Factory
- **Syncs with MongoDB**: Pulls NFT contract addresses from your existing Dopamint backend
- **Reduces costs**: Saves storage and processing by indexing only relevant data

### Contracts Indexed

1. **Factory Contract** (fixed address)
   - Events: `NFTContractCreated`, `NFTEvent`, fee updates, etc.

2. **Payment Contract** (fixed address)
   - Events: `PaymentReceived`, `Withdrawal`, etc.

3. **NFT Contracts** (dynamic addresses)
   - Events: `NFTMinted`, `NFTBurned`, `Transfer`, etc.
   - Auto-discovered from Factory's `NFTContractCreated` events
   - Synced from MongoDB every 5 minutes

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Base Chain RPC                          │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Dopamint RPC Client                          │
│           (with Contract Address Filtering)                     │
└──────────────────────────┬──────────────────────────────────────┘
                           │
          ┌────────────────┴────────────────┐
          ▼                                 ▼
┌──────────────────────┐         ┌──────────────────────┐
│   Backfill Service   │         │  Committer Service   │
│  (Historical Data)   │         │   (Live Indexing)    │
└──────────┬───────────┘         └──────────┬───────────┘
           │                                │
           ▼                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Contract Filter                            │
│  • Factory Address Filter                                       │
│  • Payment Address Filter                                       │
│  • NFT Contracts Filter (Dynamic)                               │
└──────────────────────────┬──────────────────────────────────────┘
                           │
          ┌────────────────┼────────────────┐
          ▼                ▼                ▼
   ┌──────────┐    ┌──────────┐    ┌──────────┐
   │  Kafka   │    │   S3     │    │ ClickHouse│
   │ (Stream) │    │(Storage) │    │  (Query)  │
   └──────────┘    └──────────┘    └──────────┘
          │                                │
          └────────────────┬───────────────┘
                           ▼
                  ┌──────────────────┐
                  │   Event Parser   │
                  │ (Dopamint Events)│
                  └─────────┬────────┘
                            ▼
                  ┌──────────────────┐
                  │  ClickHouse DB   │
                  │ • nft_events     │
                  │ • payment_events │
                  │ • logs           │
                  └──────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      MongoDB Integration                        │
│  • Stores NFT contract addresses                                │
│  • Syncs every 5 minutes                                        │
│  • Auto-updates indexer filter                                  │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    Auto-Discovery System                        │
│  • Listens for NFTContractCreated events                        │
│  • Automatically adds new NFT contracts to filter               │
│  • No manual intervention needed                                │
└─────────────────────────────────────────────────────────────────┘
```

---

## Key Features

### 1. **Contract Address Filtering**
- Filters logs at the RPC level using `eth_getLogs` address parameter
- Only fetches events from watched contracts
- Reduces data volume by 99%+

### 2. **Auto-Discovery**
- Monitors Factory's `NFTContractCreated` events
- Automatically adds new NFT contracts to the watch list
- No manual configuration needed for new collections

### 3. **MongoDB Synchronization**
- Pulls NFT contract addresses from your existing Dopamint backend
- Syncs every 5 minutes (configurable)
- Ensures consistency between backend and indexer

### 4. **Event Parsing**
- Parses Dopamint-specific events:
  - NFT minting, burning, transfers
  - Payment events
  - Factory events
- Stores structured data in ClickHouse

### 5. **Real-Time & Historical**
- **Backfill mode**: Index historical data from genesis
- **Committer mode**: Live indexing with reorg detection
- Seamless switching between modes

---

## Prerequisites

### Required Software
- **Docker** & **Docker Compose** (v2.0+)
- **Go** (v1.21+) - for building custom modifications
- **Node.js** (v18+) - optional, for frontend integration
- **Git**

### Infrastructure
- **Base Chain RPC**: Public or private RPC endpoint
- **MongoDB**: Existing Dopamint backend database
- **Storage**: At least 50GB for ClickHouse data

### Contract Information
- Factory contract address
- Payment contract address
- MongoDB connection details

---

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/dopamint-indexer-insight.git
cd dopamint-indexer-insight
```

### 2. Run Setup Script

```bash
chmod +x scripts/*.sh
./scripts/setup.sh
```

This will:
- Check dependencies
- Create `.env` file from template
- Start infrastructure services (ClickHouse, MongoDB, Redis, Kafka)
- Initialize database schemas

### 3. Configure Environment

Edit `.env` file and set:

```env
# Required: Your Dopamint contract addresses
DOPAMINT_FACTORY_ADDRESS=0xYourFactoryAddress
DOPAMINT_PAYMENT_ADDRESS=0xYourPaymentAddress

# Required: Base chain RPC
RPC_URL=https://base.llamarpc.com

# Required: MongoDB connection
MONGODB_URI=mongodb://your-mongodb-host:27017
MONGODB_DATABASE=dopamint
MONGODB_COLLECTION=nft_contracts
```

### 4. Update Contract Configuration

Edit `src/config/contracts.json`:

```json
{
  "contracts": {
    "factory": {
      "address": "0xYourFactoryAddress"
    },
    "payment": {
      "address": "0xYourPaymentAddress"
    }
  }
}
```

---

## Configuration

### Main Config File: `src/config/indexer_config.yaml`

#### RPC Configuration

```yaml
rpc:
  url: https://base.llamarpc.com
  chainId: 8453  # Base Mainnet

  logs:
    blocksPerRequest: 100
    useAddressFilter: true  # Enable filtering!
```

#### MongoDB Sync Settings

```yaml
mongodb:
  syncEnabled: true
  syncIntervalSeconds: 300  # 5 minutes
```

#### Filtering Settings

```yaml
dopamint:
  filtering:
    enabled: true
    mode: whitelist
    autoDiscovery: true
```

---

## Running the Indexer

### Start Infrastructure

```bash
docker-compose up -d clickhouse mongodb redis kafka
```

### Option 1: Backfill Historical Data

Index all historical data from a starting block:

```bash
# Set starting block
export BACKFILL_FROM_BLOCK=0

# Run backfill
docker-compose --profile backfill up -d indexer-backfill

# Monitor logs
docker-compose logs -f indexer-backfill
```

### Option 2: Live Indexing (Committer Mode)

Index new blocks in real-time:

```bash
export COMMITTER_IS_LIVE=true

docker-compose --profile committer up -d indexer-committer

docker-compose logs -f indexer-committer
```

### Option 3: Combined (Backfill → Live)

```bash
# 1. Backfill historical data
./scripts/start-indexer.sh backfill

# 2. Wait for backfill to complete
# (Check logs for "Backfill complete")

# 3. Start live indexing
./scripts/start-indexer.sh committer
```

---

## Custom Filtering Implementation

### How It Works

#### 1. **Contract Filter** (`src/filters/contract_filter.go`)

```go
type ContractFilter struct {
    factoryAddress    common.Address
    paymentAddress    common.Address
    nftContracts      map[common.Address]bool
}

func (cf *ContractFilter) ShouldIndexLog(address common.Address) bool {
    // Check if address is Factory, Payment, or NFT contract
    return cf.factoryAddress == address ||
           cf.paymentAddress == address ||
           cf.nftContracts[address]
}
```

#### 2. **Event Listener** (`src/filters/event_listener.go`)

```go
func (el *EventListener) ProcessLog(log types.Log) {
    // Listen for NFTContractCreated event
    if log.Topics[0] == NFTContractCreatedSignature {
        contractAddress := common.BytesToAddress(log.Topics[1].Bytes())
        el.contractFilter.AddNFTContract(contractAddress)
    }
}
```

#### 3. **RPC Integration** (`src/utils/rpc_integration.go`)

```go
func (d *DopamintRPCClient) GetFilteredLogs(ctx context.Context, fromBlock, toBlock *big.Int) ([]types.Log, error) {
    query := ethereum.FilterQuery{
        FromBlock: fromBlock,
        ToBlock:   toBlock,
        Addresses: d.addressFilter,  // Only Dopamint contracts
    }
    return d.client.FilterLogs(ctx, query)
}
```

### Integrating into Thirdweb Insight

To integrate the custom filtering into the insight codebase:

#### Step 1: Modify RPC Client

Edit `../insight/internal/rpc/rpc.go`:

```go
import "path/to/dopamint-indexer-insight/src/filters"

type RPC struct {
    // ... existing fields
    contractFilter *filters.ContractFilter
}

func (r *RPC) GetFullBlocks(ctx context.Context, from, to *big.Int) ([]*common.BlockData, error) {
    // Use filtered log fetching
    if r.contractFilter.IsEnabled() {
        addresses := r.contractFilter.GetWatchedAddresses()
        // Modify eth_getLogs call to include address filter
        query := ethereum.FilterQuery{
            FromBlock: from,
            ToBlock:   to,
            Addresses: addresses,
        }
        logs, err := r.client.FilterLogs(ctx, query)
        // ... process logs
    }
    // ... rest of the function
}
```

#### Step 2: Initialize Filter

Edit `../insight/cmd/backfill.go` and `../insight/cmd/committer.go`:

```go
import "path/to/dopamint-indexer-insight/src/filters"

func Execute() {
    // Load contract filter
    contractFilter, err := filters.NewContractFilter("config/contracts.json")
    if err != nil {
        log.Fatal(err)
    }

    // Create event listener
    eventListener := filters.NewEventListener(contractFilter, factoryAddress)

    // Initialize RPC with filter
    rpcClient := rpc.NewRPC(config.RPC.URL)
    rpcClient.SetContractFilter(contractFilter)

    // Start MongoDB sync
    mongoClient, _ := database.NewDopamintMongoClient(mongoConfig)
    go contractFilter.StartMongoDBSync(ctx, mongoClient)

    // ... rest of initialization
}
```

#### Step 3: Process Events for Auto-Discovery

In the log processing pipeline:

```go
func processLogs(logs []types.Log) {
    for _, log := range logs {
        // Let event listener check for new contracts
        eventListener.ProcessLog(log)

        // Continue with normal log processing
        // ...
    }
}
```

---

## MongoDB Integration

### Schema

Your MongoDB collection should have documents like:

```json
{
  "_id": ObjectId("..."),
  "contractAddress": "0x1234567890abcdef...",
  "collectionId": 1,
  "creator": "0xabcdef...",
  "name": "My NFT Collection",
  "symbol": "MNFT",
  "baseURI": "ipfs://...",
  "modelId": 1,
  "chainId": 8453,
  "network": "base",
  "status": "active",
  "createdAt": ISODate("2024-01-01T00:00:00Z"),
  "updatedAt": ISODate("2024-01-01T00:00:00Z")
}
```

### Auto-Sync Process

```
MongoDB                      Indexer
   │                            │
   │◄───────sync (5min)─────────│
   │                            │
   │  [List of NFT contracts]   │
   │───────────────────────────►│
   │                            │
   │                         Update
   │                         Filter
   │                            │
   ▼                            ▼
```

### Manual Sync

```bash
# Trigger immediate sync
docker-compose exec indexer-committer /app/indexer sync-contracts
```

---

## Monitoring & Troubleshooting

### Access Monitoring Tools

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **ClickHouse**: http://localhost:8123

### Key Metrics

1. **Logs Indexed**: Total events indexed
2. **Contracts Watched**: Number of contracts being monitored
3. **Blocks Processed**: Indexing progress
4. **Filtering Efficiency**: % of logs filtered out

### Common Issues

#### 1. No Logs Being Indexed

**Check:**
- Contract addresses are correct in `.env`
- RPC URL is working
- Contracts have activity on Base chain

```bash
# Test RPC connection
curl -X POST $RPC_URL \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

#### 2. MongoDB Sync Not Working

**Check:**
- MongoDB connection string
- Collection name matches
- MongoDB has data

```bash
# Test MongoDB connection
docker-compose exec mongodb mongosh $MONGODB_URI --eval "db.nft_contracts.countDocuments()"
```

#### 3. Auto-Discovery Not Adding Contracts

**Check:**
- Factory address is correct
- Backfill has processed Factory creation blocks
- Event signatures match

```bash
# Check event signature
node -e "console.log(require('ethers').utils.id('NFTContractCreated(uint256,address,address,string,string,string)'))"
```

### Logs

```bash
# View all logs
docker-compose logs -f

# View specific service
docker-compose logs -f indexer-committer

# Search logs
docker-compose logs | grep "Discovered new NFT contract"
```

---

## API Usage

### Query ClickHouse

```bash
# Total NFT mints
docker-compose exec clickhouse clickhouse-client --query "
SELECT COUNT(*) as total_mints
FROM dopamint_indexer.nft_events
WHERE event_type = 'mint'
"

# Mints by contract
docker-compose exec clickhouse clickhouse-client --query "
SELECT
    contract_address,
    COUNT(*) as mint_count,
    SUM(price) as total_volume
FROM dopamint_indexer.nft_events
WHERE event_type = 'mint'
GROUP BY contract_address
ORDER BY total_volume DESC
"

# Recent payments
docker-compose exec clickhouse clickhouse-client --query "
SELECT
    payer,
    amount,
    gen_id,
    timestamp
FROM dopamint_indexer.payment_events
WHERE event_type = 'payment'
ORDER BY timestamp DESC
LIMIT 100
"
```

### REST API (Optional)

You can build a REST API on top of ClickHouse:

```javascript
// Example API endpoint
app.get('/api/nft/:address/stats', async (req, res) => {
  const { address } = req.params;

  const query = `
    SELECT
      COUNT(*) as total_mints,
      SUM(price) as total_volume,
      AVG(price) as avg_price
    FROM dopamint_indexer.nft_events
    WHERE contract_address = '${address}' AND event_type = 'mint'
  `;

  const result = await clickhouse.query(query);
  res.json(result);
});
```

---

## Development Guide

### Project Structure

```
dopamint-indexer-insight/
├── src/
│   ├── config/
│   │   ├── contracts.json           # Contract addresses
│   │   └── indexer_config.yaml      # Main configuration
│   ├── contracts/
│   │   └── abis.ts                  # Contract ABIs
│   ├── database/
│   │   └── mongodb_client.go        # MongoDB integration
│   ├── filters/
│   │   ├── contract_filter.go       # Address filtering
│   │   └── event_listener.go        # Auto-discovery
│   └── utils/
│       └── rpc_integration.go       # RPC client wrapper
├── scripts/
│   ├── setup.sh                     # Initial setup
│   ├── init-databases.sh            # DB initialization
│   └── start-indexer.sh             # Start script
├── docs/
│   └── README.md                    # This file
├── docker-compose.yml               # Infrastructure
└── .env.example                     # Environment template
```

### Building Custom Modifications

1. **Modify Go Code**

```bash
cd ../insight
go mod tidy
go build -o indexer .
```

2. **Test Locally**

```bash
./indexer backfill --config ../dopamint-indexer-insight/src/config/indexer_config.yaml
```

3. **Build Docker Image**

```bash
docker build -t dopamint-indexer:latest .
```

### Adding New Event Types

1. **Update ABI** in `src/contracts/abis.ts`
2. **Add Parser** in `src/filters/event_listener.go`
3. **Update Schema** in `scripts/init-databases.sh`
4. **Rebuild** and restart indexer

---

## Performance Optimization

### Recommended Settings

#### For Small Deployments (<10 NFT contracts)
```yaml
rpc:
  logs:
    blocksPerRequest: 100
backfill:
  batchSize: 1000
  concurrency: 2
```

#### For Medium Deployments (10-100 NFT contracts)
```yaml
rpc:
  logs:
    blocksPerRequest: 50
backfill:
  batchSize: 500
  concurrency: 4
```

#### For Large Deployments (100+ NFT contracts)
```yaml
rpc:
  logs:
    blocksPerRequest: 25
backfill:
  batchSize: 250
  concurrency: 8
```

### Storage Estimates

- **Per 1M blocks**: ~10GB ClickHouse storage
- **Per 1M logs**: ~5GB ClickHouse storage
- **MongoDB**: <100MB for contract metadata

---

## Production Deployment

### Checklist

- [ ] Use dedicated RPC endpoint (not public)
- [ ] Set up ClickHouse cluster for redundancy
- [ ] Configure automated backups
- [ ] Set up monitoring alerts
- [ ] Use Redis for distributed locking
- [ ] Configure Kafka for scalability
- [ ] Set up log aggregation (ELK/Loki)
- [ ] Implement API rate limiting
- [ ] Set up CI/CD pipeline

### Kubernetes Deployment

Helm chart coming soon...

---

## FAQ

**Q: How much does it cost to run?**
A: Depends on RPC provider. With filtering, you'll use ~90% less RPC calls than full indexing.

**Q: Can I index multiple chains?**
A: Yes, run separate instances with different configs.

**Q: What if I miss a block?**
A: The committer detects gaps and refetches missing blocks.

**Q: Can I query historical data?**
A: Yes, ClickHouse retains all historical data.

**Q: How do I backup data?**
A: ClickHouse supports backup to S3. MongoDB Atlas has built-in backups.

---

## Support & Contributing

- **Issues**: https://github.com/your-org/dopamint-indexer-insight/issues
- **Docs**: https://docs.dopamint.io/indexer
- **Discord**: https://discord.gg/dopamint

---

## License

MIT License - see LICENSE file

---

**Built with ❤️ by the Dopamint team**
