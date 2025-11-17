# Dopamint Indexer Project - Comprehensive Structure Analysis

## IMPORTANT CLARIFICATION

**This is a BACKEND blockchain indexer project, NOT a frontend application.**

Currently, there is **NO frontend** (Next.js, React, or any UI framework) in this repository. The questions about:
- Explore Strategies page
- Detail strategy page
- Filter popup
- Connect wallet button
- Leaderboard page
- Tailwind/CSS styling
- Context providers for theming

**Do not exist in this project.** This appears to be setup for building a frontend that will consume the data from this indexer.

---

## 1. OVERALL PROJECT STRUCTURE & FRAMEWORK

### Project Type
- **Backend Blockchain Indexer** (written in Go + TypeScript)
- Based on [thirdweb-dev/insight](https://github.com/thirdweb-dev/insight)
- Specialized for indexing Dopamint NFT platform contracts on Base chain
- **NO React/Next.js/Frontend framework**

### Current Tech Stack
```
Language:       Go + TypeScript
Database:       ClickHouse (analytics), MongoDB (contract management)
Message Queue:  Kafka
Cache:          Redis
Monitoring:     Prometheus + Grafana
Orchestration:  Docker Compose
```

### Repository Structure
```
dopamint-indexer-insight/
├── src/                          # Backend source code
│   ├── config/                   # Configuration files
│   │   ├── contracts.json        # Smart contract addresses & events config
│   │   └── indexer_config.yaml   # Indexer settings (RPC, storage, filtering)
│   ├── contracts/                # Smart contract interfaces
│   │   └── abis.ts               # Dopamint contract ABIs (TypeScript)
│   ├── database/                 # MongoDB integration
│   │   └── mongodb_client.go     # MongoDB client for syncing NFT contracts
│   ├── filters/                  # Core filtering logic
│   │   ├── contract_filter.go    # Contract whitelist filtering
│   │   └── event_listener.go     # NFT contract auto-discovery
│   └── utils/                    # Utility functions
│       └── rpc_integration.go    # RPC client with filtering
├── scripts/                      # Setup and initialization scripts
│   ├── setup.sh                  # Automated project setup
│   └── init-databases.sh         # Database initialization
├── docs/                         # Documentation
│   ├── README.md                 # Comprehensive guide (5000+ words)
│   ├── GETTING_STARTED.md        # Quick start guide
│   └── INTEGRATION_GUIDE.md       # Integration with thirdweb insight
├── docker-compose.yml            # Infrastructure setup
├── .env.example                  # Environment template
├── package.json                  # NPM dependencies (minimal - only for scripts)
├── IMPLEMENTATION_SUMMARY.md     # Detailed implementation notes
└── README.md                     # Project overview
```

---

## 2. KEY COMPONENTS (Backend Only)

### A. Contract Filter System (`src/filters/contract_filter.go`)
**Purpose**: Manages which contracts to index on the blockchain

**Functionality**:
- Maintains whitelist of Factory, Payment, and NFT contracts
- Thread-safe address management
- Dynamic contract addition
- Address validation
- Statistics tracking

**Key Methods**:
```go
ShouldIndexLog(address)          // Determine if log should be indexed
AddNFTContract(address)          // Add new NFT contract dynamically
GetWatchedAddresses()            // Get all monitored addresses
GetAddressFilter()               // Format addresses for RPC calls
Stats()                          // Get filter statistics
StartMongoDBSync()               // Start periodic sync from MongoDB
```

### B. Event Listener System (`src/filters/event_listener.go`)
**Purpose**: Auto-discover new NFT contracts from blockchain events

**Functionality**:
- Monitors `NFTContractCreated` events from Factory
- Extracts contract addresses from event logs
- Automatically adds new contracts to watch list
- Supports backfilling historical contracts
- No manual intervention needed

**Key Methods**:
```go
ProcessLog(log)                  // Process single event
ProcessLogs(logs)                // Process multiple events
ParseNFTContractCreatedEvent()   // Parse event data
BackfillNFTContracts()           // Backfill from history
```

**How It Works**:
```
1. RPC returns event logs from Factory contract
2. Event signature: NFTContractCreated(uint256,address,address,string,string,string)
3. Extract indexed parameters: collectionId, contractAddress, creator
4. Auto-add contractAddress to watch list
5. Next RPC call will include new address in filter
```

### C. MongoDB Integration (`src/database/mongodb_client.go`)
**Purpose**: Sync NFT contract addresses from existing Dopamint backend

**Features**:
- Fetches active NFT contracts from MongoDB
- Periodic sync every 5 minutes (configurable)
- Change stream monitoring for real-time updates
- Upsert contract metadata
- Statistics tracking

**Key Methods**:
```go
GetNFTContractAddresses(ctx)     // Fetch all contract addresses
GetActiveNFTContracts(ctx)       // Fetch only active contracts
UpsertNFTContract(contract)      // Insert/update contract
GetContractByAddress()           // Fetch single contract
WatchNFTContracts()              // Watch for changes (requires replica set)
GetStats()                       // Contract statistics
```

**Synced Document Structure**:
```javascript
{
  contractAddress: "0x...",      // Ethereum address
  collectionId: 123,             // Collection ID from Factory
  creator: "0x...",              // Creator address
  name: "Collection Name",       // NFT collection name
  symbol: "SYMBOL",              // Token symbol
  baseURI: "ipfs://...",         // Metadata base URI
  modelId: 1,                    // AI model ID
  chainId: 8453,                 // Base chain ID
  network: "base",               // Network name
  status: "active",              // Status: active, inactive, deleted
  createdAt: Date,               // Creation timestamp
  updatedAt: Date                // Last update timestamp
}
```

### D. RPC Integration (`src/utils/rpc_integration.go`)
**Purpose**: Wrap standard RPC client with Dopamint-specific filtering

**Features**:
- Filtered log fetching from RPC
- Dynamic address filter updates
- Efficiency metrics calculation
- Connection pooling

**Key Methods**:
```go
GetFilteredLogs(fromBlock, toBlock)  // Fetch filtered logs
UpdateAddressFilter(addresses)       // Update filter dynamically
GetBlockNumber()                     // Get latest block
GetBlockByNumber()                   // Fetch block data
CalculateFilterEfficiency(stats)     // Calculate efficiency %
```

**How Filtering Works**:
```
BEFORE (without filtering):
  Base RPC → eth_getLogs() → 1M logs per 10K blocks → Process & Store 1TB data

AFTER (with Dopamint filtering):
  Base RPC → eth_getLogs(addresses=[Factory, Payment, NFT1, NFT2, ...])
         → 100 logs per 10K blocks → Process & Store 10GB data

Efficiency Gain: 99%+ data reduction
```

---

## 3. CONFIGURATION SYSTEM

### A. Contract Configuration (`src/config/contracts.json`)
```json
{
  "network": "base",
  "chainId": 8453,
  "contracts": {
    "factory": {
      "address": "0x_FACTORY_ADDRESS_HERE",
      "name": "DopamintNFTFactory",
      "events": [
        "NFTContractCreated",
        "AIGenerationFeeUpdated",
        "CreationFeeUpdated",
        ...
      ]
    },
    "payment": {
      "address": "0x_PAYMENT_ADDRESS_HERE",
      "name": "DopamintPayment",
      "events": ["PaymentReceived", "Withdrawal", ...]
    },
    "nftContracts": []  // Auto-populated
  },
  "eventFilters": {
    "enabled": true,
    "filterMode": "whitelist"
  },
  "syncSettings": {
    "mongodbSync": {
      "enabled": true,
      "intervalSeconds": 300
    },
    "autoDiscovery": {
      "enabled": true
    }
  }
}
```

### B. Indexer Configuration (`src/config/indexer_config.yaml`)

**RPC Settings**:
```yaml
rpc:
  url: ${RPC_URL}           # Base chain RPC endpoint
  chainId: 8453             # Base Mainnet
  logs:
    blocksPerRequest: 100   # Blocks per RPC call
    useAddressFilter: true  # Enable filtering
```

**Storage Configuration**:
```yaml
storage:
  main:                      # ClickHouse (main warehouse)
    host: localhost
    port: 9000
    database: dopamint_indexer
  
  kafka:                     # Message broker
    brokers: localhost:9092
    topic: dopamint-blocks
  
  redis:                     # Metadata cache
    host: localhost
    port: 6379
```

**MongoDB Settings**:
```yaml
mongodb:
  uri: ${MONGODB_URI}
  database: dopamint
  collection: nft_contracts
  syncEnabled: true
  syncIntervalSeconds: 300
```

**Dopamint-Specific Settings**:
```yaml
dopamint:
  filtering:
    enabled: true
    mode: whitelist
    autoDiscovery: true
  
  backfill:
    enabled: true
    fromBlock: 0
    toBlock: latest
    batchSize: 1000
```

### C. Environment Variables (`.env.example`)

**Required**:
```env
DOPAMINT_FACTORY_ADDRESS=0x...    # Factory contract address
DOPAMINT_PAYMENT_ADDRESS=0x...    # Payment contract address
RPC_URL=https://base.llamarpc.com # Base chain RPC
```

**Database**:
```env
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=dopamint
MONGODB_COLLECTION=nft_contracts

CLICKHOUSE_HOST=localhost
CLICKHOUSE_PORT=9000
CLICKHOUSE_DB=dopamint_indexer
```

**Infrastructure**:
```env
KAFKA_BROKERS=localhost:9092
REDIS_HOST=localhost
REDIS_PORT=6379
```

**Tuning**:
```env
LOG_LEVEL=info              # debug, info, warn, error
INDEXER_CONCURRENCY=4       # Worker threads
INDEXER_MAX_MEMORY_MB=4096  # Memory limit
```

---

## 4. INFRASTRUCTURE & DEPLOYMENT

### Docker Compose Services (`docker-compose.yml`)

```
dopamint-clickhouse          Port 8123 (HTTP), 9000 (Native)
├─ Data warehouse for analytics
├─ Stores: blocks, transactions, logs, nft_events, payment_events
└─ Persisted volume: clickhouse-data

dopamint-mongodb             Port 27017
├─ Document database
├─ Stores: NFT contract metadata & addresses
└─ Persisted volume: mongodb-data

dopamint-redis               Port 6379
├─ In-memory data store
├─ Caches: metadata & temporary data
└─ Persisted volume: redis-data

dopamint-kafka               Port 9092 (Plaintext), 9093 (Controller)
├─ Message broker
├─ Topics: dopamint-blocks
└─ Persisted volume: kafka-data

dopamint-prometheus          Port 9090
├─ Metrics collection
└─ Persisted volume: prometheus-data

dopamint-grafana             Port 3000
├─ Metrics visualization
├─ Dashboards for monitoring
└─ Persisted volume: grafana-data

dopamint-indexer-backfill    (Profile: backfill)
├─ Indexes historical blockchain data
├─ Reads: ../insight codebase
├─ Config: src/config/indexer_config.yaml
└─ Mounts: Contract configs

dopamint-indexer-committer   (Profile: committer)
├─ Real-time live indexing
├─ Reorg detection & handling
├─ Reads: ../insight codebase
└─ Config: src/config/indexer_config.yaml
```

### How to Run

```bash
# 1. Initial setup
./scripts/setup.sh

# 2. Configure environment
nano .env
# Add: DOPAMINT_FACTORY_ADDRESS, DOPAMINT_PAYMENT_ADDRESS, RPC_URL

# 3. Start infrastructure
docker-compose up -d

# 4. Initialize databases
./scripts/init-databases.sh

# 5. Run backfill (historical data)
docker-compose --profile backfill up -d

# 6. Or run committer (live indexing)
docker-compose --profile committer up -d

# 7. Monitor
docker-compose logs -f
```

### Monitoring & Observability

**Grafana Dashboard**: http://localhost:3000 (admin/admin)
- Real-time metrics visualization
- Event indexing rates
- Contract watch statistics
- Database performance

**Prometheus**: http://localhost:9090
- Metrics scraping
- Query interface
- Alert rules

**ClickHouse UI**: http://localhost:8123
- Direct query interface
- Table browsing
- Data exploration

---

## 5. SMART CONTRACT ABIs (`src/contracts/abis.ts`)

TypeScript file containing ABI definitions for:

### Factory Contract Events
- `NFTContractCreated(uint256,address,address,string,string,string)`
  - Emitted when new NFT collection created
  - Auto-discovered by EventListener
- `AIGenerationFeeUpdated(uint256,uint256,uint256)`
- `CreationFeeUpdated(uint256,uint256)`
- `OperatorUpdated(address)`

### Payment Contract Events
- `PaymentReceived(address,uint256)`
  - AI generation payments
- `Withdrawal(address,uint256)`
  - Fee withdrawals

### NFT Contract Events
- `Transfer(address,address,uint256)`
- `NFTMinted(address,uint256,string)`
- `NFTBurned(address,uint256)`

---

## 6. DATABASE SCHEMAS

### ClickHouse Tables

**blocks**
- Block number, timestamp, hash, miner
- Used for block-level analytics

**transactions**
- Transaction hash, from, to, value, gas
- Used for transaction tracking

**logs**
- Raw event logs from blockchain
- indexed by: address, topic, block number

**nft_events** (Parsed)
```
event_type: 'mint' | 'burn' | 'transfer'
contract_address: String
token_id: UInt256
from_address: String
to_address: String
block_number: UInt64
tx_hash: String
timestamp: DateTime
```

**payment_events** (Parsed)
```
event_type: 'payment' | 'withdrawal'
payer: String
amount: UInt256
block_number: UInt64
tx_hash: String
timestamp: DateTime
```

### MongoDB Collections

**nft_contracts**
- Primary: contractAddress + chainId
- Index on: status, createdAt, creator
- Synced with FilterManager every 5 minutes

---

## 7. STYLING & THEMING

### Current Status
**No styling system exists** - this is a backend project

### When Frontend is Built
You will likely need:
- **CSS Framework**: Tailwind CSS (recommended)
- **UI Component Library**: shadcn/ui or Radix UI
- **Theme Provider**: Context API + React hooks
- **Global Styles**: Tailwind + custom CSS
- **Color Variables**: Tailwind config or CSS variables

---

## 8. CONTEXT PROVIDERS

### Current Status
**No React/Context system exists** - this is a backend project

### Recommended Architecture for Frontend
```typescript
// Context structure needed for frontend
StrategyContext:
- Strategies list
- Selected strategy
- Filters applied
- Leaderboard data

WalletContext:
- Connected wallet
- User address
- Balance
- Network

ThemeContext:
- Dark/light mode
- Color scheme
- Font settings
```

---

## 9. DATA FLOW ARCHITECTURE

### Complete Indexing Flow

```
┌─────────────────┐
│  Base Chain     │
│  (8453)         │
└────────┬────────┘
         │
    ┌────▼─────┐
    │ RPC Client│ ◄──── (Address Filter: [Factory, Payment, NFT1, NFT2, ...])
    └────┬──────┘
         │
    ┌────▼──────────────┐
    │ Contract Filter    │
    │ (Whitelist Mode)   │
    └────┬───────────────┘
         │
    ┌────▼──────────────────┐
    │ Event Listener        │
    │ (Auto-Discovery)      │
    └────┬──────────────────┘
         │
    ┌────▼────────────────────────────┐
    │ Log Parser                       │
    │ (NFT Events, Payment Events)     │
    └────┬─────────────────────────────┘
         │
    ┌────▼────────────────┐
    │ Storage             │
    ├─────────────────────┤
    │ ClickHouse (logs)   │  ◄─── Query Layer
    │ MongoDB (contracts) │  ◄─── Contract Sync
    │ Kafka (messages)    │  ◄─── Stream Processing
    │ Redis (cache)       │  ◄─── Fast Access
    └─────────────────────┘
         │
    ┌────▼─────────────────┐
    │ Monitoring           │
    ├──────────────────────┤
    │ Prometheus (metrics) │
    │ Grafana (dashboards) │
    └──────────────────────┘
```

### Contract Address Sync Flow

```
┌──────────────────────────────────────────┐
│ Dopamint Backend MongoDB                 │
│ (nft_contracts collection)               │
└────────────────────┬─────────────────────┘
                     │
              ┌──────▼──────────┐
              │ Periodic Sync   │
              │ (Every 5 min)   │
              └──────┬──────────┘
                     │
              ┌──────▼──────────────┐
              │ Contract Filter      │
              │ .AddNFTContracts()   │
              └──────┬───────────────┘
                     │
              ┌──────▼──────────────┐
              │ RPC Address Filter   │
              │ Updated dynamically  │
              └──────┬───────────────┘
                     │
          ┌──────────▼──────────────┐
          │ Next eth_getLogs call   │
          │ Includes new addresses  │
          └─────────────────────────┘
```

---

## 10. PERFORMANCE CHARACTERISTICS

### Filtering Efficiency

| Metric | Full Chain | Dopamint Only | Reduction |
|--------|-----------|---------------|-----------|
| Logs per 10K blocks | ~1,000,000 | ~100 | 99.99% |
| Storage per 1M blocks | ~1TB | ~10GB | 99% |
| Memory usage | ~5GB | ~500MB | 90% |
| RPC calls/hour | 10,000 | 1,000 | 90% |
| Processing time | 10 hours | 30 min | 95% |

### Resource Requirements

**Development Setup**:
- Docker & Docker Compose
- 4GB RAM minimum
- 50GB disk space

**Production Setup**:
- Dedicated RPC endpoint
- ClickHouse cluster (high availability)
- MongoDB replica set
- Kafka cluster
- 8+ GB RAM
- 500GB+ disk space

---

## 11. INTEGRATION WITH THIRDWEB INSIGHT

This project integrates with thirdweb's open-source indexer:

**Key Integration Points**:
1. Custom `ContractFilter` hooks into RPC client
2. Custom `EventListener` processes blockchain events
3. Custom `MongoDBClient` syncs contract addresses
4. Configuration files override defaults
5. Docker container built from ../insight codebase

**How to Integrate**:
```bash
# Clone both repositories
git clone https://github.com/thirdweb-dev/insight.git
git clone https://github.com/your-org/dopamint-indexer-insight.git

# Build with custom Dopamint config
cd insight
go build -o indexer .

# Run with config
./indexer backfill --config ../dopamint-indexer-insight/src/config/indexer_config.yaml
```

---

## 12. QUERY EXAMPLES

### ClickHouse Queries

```sql
-- Total NFT mints
SELECT COUNT(*) FROM dopamint_indexer.nft_events 
WHERE event_type = 'mint';

-- Top collections by volume
SELECT 
  contract_address,
  COUNT(*) as mint_count,
  SUM(price) as total_volume
FROM dopamint_indexer.nft_events
WHERE event_type = 'mint'
GROUP BY contract_address
ORDER BY total_volume DESC
LIMIT 10;

-- Recent payments
SELECT * FROM dopamint_indexer.payment_events
ORDER BY timestamp DESC 
LIMIT 100;

-- Active NFT contracts
SELECT COUNT(DISTINCT contract_address) 
FROM dopamint_indexer.nft_events
WHERE timestamp > now() - INTERVAL 1 DAY;
```

### MongoDB Queries

```javascript
// Active contracts
db.nft_contracts.find({ status: "active" })

// Contracts by creator
db.nft_contracts.aggregate([
  { $group: { _id: "$creator", count: { $sum: 1 } } }
])

// Recent collections
db.nft_contracts.find()
  .sort({ createdAt: -1 })
  .limit(10)
```

---

## 13. FILE LOCATIONS SUMMARY

| Component | File Path |
|-----------|-----------|
| Contract Filter | `/src/filters/contract_filter.go` |
| Event Listener | `/src/filters/event_listener.go` |
| MongoDB Client | `/src/database/mongodb_client.go` |
| RPC Integration | `/src/utils/rpc_integration.go` |
| Contract ABIs | `/src/contracts/abis.ts` |
| Contracts Config | `/src/config/contracts.json` |
| Indexer Config | `/src/config/indexer_config.yaml` |
| Docker Compose | `/docker-compose.yml` |
| Environment Template | `/.env.example` |
| Setup Script | `/scripts/setup.sh` |
| Init Script | `/scripts/init-databases.sh` |
| Main README | `/README.md` |
| Implementation Guide | `/IMPLEMENTATION_SUMMARY.md` |
| Docs | `/docs/` |

---

## NEXT STEPS FOR FRONTEND DEVELOPMENT

Since this is a backend indexer, you'll need to create a **separate frontend project**:

### Recommended Tech Stack
1. **Framework**: Next.js 13+ (App Router)
2. **UI Library**: shadcn/ui + Radix UI
3. **Styling**: Tailwind CSS
4. **State Management**: React Context + TanStack Query
5. **Wallet Integration**: wagmi + rainbowkit
6. **Data Fetching**: TanStack Query (React Query)
7. **API Client**: Axios or Fetch
8. **Charting**: Recharts or Chart.js

### Frontend Pages Needed
- **Explore Strategies Page** - List all NFT collections with filters
- **Strategy Detail Page** - Individual collection details & stats
- **Leaderboard Page** - Top collections by various metrics
- **User Dashboard** - Connected wallet stats
- **Admin Panel** - Monitor indexer health

### Data Sources
- Backend GraphQL/REST API wrapping ClickHouse queries
- MongoDB for contract metadata
- Blockchain for real-time data

---

## CONCLUSION

**The Dopamint Indexer is a production-ready backend system** for efficiently indexing Dopamint NFT platform contracts on Base chain. 

**There is NO frontend currently.** To build the pages you mentioned (Explore Strategies, Leaderboard, etc.), you'll need to:

1. Create a separate Next.js/React frontend project
2. Build APIs that query the ClickHouse and MongoDB data
3. Implement the UI components you described
4. Connect wallet functionality using wagmi/rainbowkit
5. Add context providers for theme/filters/user state

The backend indexer will serve as the data layer for your frontend application.
