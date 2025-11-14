# Getting Started with Dopamint Indexer

This guide will help you get the Dopamint Indexer up and running in 15 minutes.

## Prerequisites Checklist

Before you begin, ensure you have:

- [ ] Docker and Docker Compose installed
- [ ] Access to a Base chain RPC endpoint
- [ ] Your Dopamint Factory contract address
- [ ] Your Dopamint Payment contract address
- [ ] MongoDB connection details (if using existing backend)
- [ ] At least 50GB free disk space

## Step 1: Clone the Repository (2 min)

```bash
git clone https://github.com/your-org/dopamint-indexer-insight.git
cd dopamint-indexer-insight
```

## Step 2: Initial Setup (3 min)

Run the automated setup script:

```bash
chmod +x scripts/*.sh
./scripts/setup.sh
```

This will:
- Create `.env` file from template
- Check Docker installation
- Create necessary directories
- Start infrastructure services (ClickHouse, MongoDB, Redis, Kafka)
- Initialize database schemas

## Step 3: Configure Your Contracts (2 min)

### Edit `.env` file

```bash
nano .env
```

**Required settings:**

```env
# Your Dopamint contract addresses (REQUIRED)
DOPAMINT_FACTORY_ADDRESS=0xYourFactoryContractAddress
DOPAMINT_PAYMENT_ADDRESS=0xYourPaymentContractAddress

# Base chain RPC endpoint (REQUIRED)
RPC_URL=https://base.llamarpc.com
# Or use your own: https://base-mainnet.g.alchemy.com/v2/YOUR_API_KEY

# MongoDB connection (if using existing backend)
MONGODB_URI=mongodb://localhost:27017
MONGODB_DATABASE=dopamint
MONGODB_COLLECTION=nft_contracts
```

### Edit contract config

```bash
nano src/config/contracts.json
```

Update the addresses:

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

## Step 4: Verify Infrastructure (2 min)

Check that all services are running:

```bash
docker-compose ps
```

You should see:
- ✅ clickhouse - running
- ✅ mongodb - running
- ✅ redis - running
- ✅ kafka - running

Test ClickHouse:
```bash
docker-compose exec clickhouse clickhouse-client --query "SELECT 1"
```

Test MongoDB:
```bash
docker-compose exec mongodb mongosh --eval "db.adminCommand('ping')"
```

## Step 5: Choose Your Indexing Mode (1 min)

You have two options:

### Option A: Backfill Historical Data

Start from a specific block and index all historical data:

```bash
# Edit backfill settings
export BACKFILL_FROM_BLOCK=0  # Or your Factory deployment block

# Start backfill
docker-compose --profile backfill up -d indexer-backfill
```

**When to use:**
- First time setup
- Need historical data
- Want to build complete database

### Option B: Live Indexing Only

Start indexing from the current block:

```bash
export COMMITTER_IS_LIVE=true

docker-compose --profile committer up -d indexer-committer
```

**When to use:**
- Only care about new data
- Testing the setup
- Already have historical data

### Option C: Both (Recommended)

1. First run backfill until complete
2. Then start committer for live data

## Step 6: Monitor Progress (5 min)

### View Logs

```bash
# All logs
docker-compose logs -f

# Specific service
docker-compose logs -f indexer-backfill
docker-compose logs -f indexer-committer
```

### What to Look For

✅ **Good Signs:**
```
[ContractFilter] Contract filter initialized: map[enabled:true ...]
[ContractFilter] Loaded 5 NFT contracts from MongoDB
[RPC] Fetched 150 logs from 7 contracts (blocks 1000-2000)
[EventListener] Discovered new NFT contract: 0x...
```

❌ **Warning Signs:**
```
Failed to connect to RPC
MongoDB connection refused
Error: contract not found
```

### Check Data in ClickHouse

```bash
# Count indexed blocks
docker-compose exec clickhouse clickhouse-client --query "
SELECT COUNT(*) as blocks
FROM dopamint_indexer.blocks
"

# Count indexed logs
docker-compose exec clickhouse clickhouse-client --query "
SELECT COUNT(*) as logs
FROM dopamint_indexer.logs
"

# Count NFT events
docker-compose exec clickhouse clickhouse-client --query "
SELECT event_type, COUNT(*) as count
FROM dopamint_indexer.nft_events
GROUP BY event_type
"
```

## Step 7: Access Monitoring Dashboards

Open your browser:

1. **Grafana**: http://localhost:3000
   - Login: admin / admin
   - View indexing metrics

2. **Prometheus**: http://localhost:9090
   - View raw metrics
   - Check targets health

3. **ClickHouse**: http://localhost:8123
   - Direct database access

## Quick Verification Test

Run this test to verify everything is working:

```bash
# 1. Check RPC connection
curl -X POST $RPC_URL -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# 2. Check contract filter
docker-compose logs indexer-backfill | grep "Contract filter initialized"

# 3. Check if logs are being indexed
docker-compose exec clickhouse clickhouse-client --query "
SELECT COUNT(*) FROM dopamint_indexer.logs WHERE created_at > now() - INTERVAL 5 MINUTE
"

# 4. Check MongoDB sync (if enabled)
docker-compose logs | grep "Synced.*NFT contracts from MongoDB"
```

## Common First-Time Issues

### Issue 1: Docker Not Running

**Error:** `Cannot connect to the Docker daemon`

**Solution:**
```bash
# Start Docker
sudo systemctl start docker

# Or on Mac
open -a Docker
```

### Issue 2: Port Already in Use

**Error:** `port is already allocated`

**Solution:**
```bash
# Check what's using the port
lsof -i :9000  # ClickHouse
lsof -i :27017  # MongoDB

# Stop the service or change ports in docker-compose.yml
```

### Issue 3: RPC Connection Failed

**Error:** `Failed to connect to RPC`

**Solution:**
1. Check RPC URL is correct
2. Test RPC endpoint:
```bash
curl -X POST $RPC_URL -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
```
3. Try a different RPC provider

### Issue 4: No Logs Being Indexed

**Possible Causes:**
1. Contract addresses are wrong
2. Contracts have no on-chain activity yet
3. Start block is too high

**Solution:**
```bash
# Verify contract addresses on Base explorer
open "https://basescan.org/address/$DOPAMINT_FACTORY_ADDRESS"

# Check contract has transactions
curl -X POST $RPC_URL -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_getCode\",\"params\":[\"$DOPAMINT_FACTORY_ADDRESS\",\"latest\"],\"id\":1}"
```

## Next Steps

Now that your indexer is running:

1. **Monitor Performance**
   - Check Grafana dashboards
   - Monitor disk space usage
   - Track indexing speed

2. **Configure Alerts**
   - Set up Prometheus alerts
   - Monitor for errors
   - Track lag behind chain tip

3. **Build API Layer**
   - Create REST endpoints
   - Add GraphQL support
   - Implement caching

4. **Integrate with Frontend**
   - Query NFT events
   - Display collection stats
   - Show real-time activity

## Helpful Commands

```bash
# Start all services
docker-compose up -d

# Stop all services
docker-compose down

# View logs
docker-compose logs -f

# Restart a service
docker-compose restart indexer-committer

# Check service status
docker-compose ps

# Clean everything (WARNING: deletes data)
docker-compose down -v

# Update containers
docker-compose pull
docker-compose up -d
```

## Getting Help

If you're stuck:

1. **Check the logs** first:
   ```bash
   docker-compose logs -f
   ```

2. **Review documentation:**
   - [Main README](../README.md)
   - [Integration Guide](INTEGRATION_GUIDE.md)
   - [Troubleshooting](../README.md#troubleshooting)

3. **Ask for help:**
   - GitHub Issues
   - Discord community
   - Email support

## Advanced Configuration

Once the basics are working, explore:

- [Performance Tuning](../src/config/indexer_config.yaml)
- [Custom Event Parsing](INTEGRATION_GUIDE.md)
- [Production Deployment](../README.md#production-deployment)
- [Scaling Strategies](../README.md#scaling)

---

## Congratulations! 🎉

Your Dopamint Indexer is now running! You're indexing blockchain data efficiently and can start building amazing features on top of it.

**Questions?** Open an issue or join our Discord!
