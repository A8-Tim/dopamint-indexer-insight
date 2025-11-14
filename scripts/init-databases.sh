#!/bin/bash

# Database Initialization Script
# Initializes ClickHouse and MongoDB with required schemas

set -e

echo "======================================"
echo "Initializing Databases"
echo "======================================"
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Initialize ClickHouse
echo "Initializing ClickHouse schema..."
docker-compose exec -T clickhouse clickhouse-client --query "
CREATE DATABASE IF NOT EXISTS dopamint_indexer;

-- Blocks table
CREATE TABLE IF NOT EXISTS dopamint_indexer.blocks (
    chain_id UInt256,
    block_number UInt256,
    block_hash String,
    parent_hash String,
    timestamp DateTime64(3),
    miner String,
    gas_limit UInt256,
    gas_used UInt256,
    base_fee_per_gas Nullable(UInt256),
    transactions_count UInt32,
    created_at DateTime64(3) DEFAULT now64()
) ENGINE = ReplacingMergeTree()
ORDER BY (chain_id, block_number)
SETTINGS index_granularity = 8192;

-- Transactions table
CREATE TABLE IF NOT EXISTS dopamint_indexer.transactions (
    chain_id UInt256,
    tx_hash String,
    block_number UInt256,
    block_hash String,
    from_address String,
    to_address Nullable(String),
    value UInt256,
    gas_price UInt256,
    gas_limit UInt256,
    gas_used Nullable(UInt256),
    nonce UInt256,
    transaction_index UInt32,
    input String,
    status Nullable(UInt8),
    created_at DateTime64(3) DEFAULT now64()
) ENGINE = ReplacingMergeTree()
ORDER BY (chain_id, block_number, transaction_index)
SETTINGS index_granularity = 8192;

-- Logs table (Events)
CREATE TABLE IF NOT EXISTS dopamint_indexer.logs (
    chain_id UInt256,
    block_number UInt256,
    block_hash String,
    tx_hash String,
    tx_index UInt32,
    log_index UInt32,
    address String,
    topic0 Nullable(String),
    topic1 Nullable(String),
    topic2 Nullable(String),
    topic3 Nullable(String),
    data String,
    removed Bool,
    created_at DateTime64(3) DEFAULT now64(),

    -- Indexes for faster queries
    INDEX idx_address address TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_topic0 topic0 TYPE bloom_filter(0.01) GRANULARITY 1
) ENGINE = ReplacingMergeTree()
ORDER BY (chain_id, block_number, log_index)
SETTINGS index_granularity = 8192;

-- Dopamint NFT Events (Parsed)
CREATE TABLE IF NOT EXISTS dopamint_indexer.nft_events (
    chain_id UInt256,
    block_number UInt256,
    tx_hash String,
    log_index UInt32,
    event_type String,  -- 'mint', 'burn', 'transfer', 'created'
    contract_address String,
    from_address Nullable(String),
    to_address Nullable(String),
    token_id Nullable(UInt256),
    generate_id Nullable(UInt256),
    price Nullable(UInt256),
    protocol_fee Nullable(UInt256),
    creator_fee Nullable(UInt256),
    timestamp DateTime64(3),
    created_at DateTime64(3) DEFAULT now64(),

    INDEX idx_contract contract_address TYPE bloom_filter(0.01) GRANULARITY 1,
    INDEX idx_event_type event_type TYPE bloom_filter(0.01) GRANULARITY 1
) ENGINE = ReplacingMergeTree()
ORDER BY (chain_id, block_number, log_index)
SETTINGS index_granularity = 8192;

-- Payment Events (Parsed)
CREATE TABLE IF NOT EXISTS dopamint_indexer.payment_events (
    chain_id UInt256,
    block_number UInt256,
    tx_hash String,
    log_index UInt32,
    event_type String,  -- 'payment', 'withdrawal'
    payer Nullable(String),
    amount UInt256,
    gen_id Nullable(UInt256),
    timestamp DateTime64(3),
    created_at DateTime64(3) DEFAULT now64()
) ENGINE = ReplacingMergeTree()
ORDER BY (chain_id, block_number, log_index)
SETTINGS index_granularity = 8192;
"

echo -e "${GREEN}✓ ClickHouse schema initialized${NC}"
echo ""

# Initialize MongoDB
echo "Initializing MongoDB collections..."
docker-compose exec -T mongodb mongosh dopamint --eval "
// Create NFT contracts collection
db.createCollection('nft_contracts');

// Create indexes
db.nft_contracts.createIndex({ 'contractAddress': 1, 'chainId': 1 }, { unique: true });
db.nft_contracts.createIndex({ 'creator': 1 });
db.nft_contracts.createIndex({ 'status': 1 });
db.nft_contracts.createIndex({ 'createdAt': -1 });
db.nft_contracts.createIndex({ 'collectionId': 1 });

// Insert sample NFT contract (if needed)
// db.nft_contracts.insertOne({
//   contractAddress: '0x...',
//   collectionId: NumberLong(1),
//   creator: '0x...',
//   name: 'Sample Collection',
//   symbol: 'SAMPLE',
//   baseURI: 'ipfs://...',
//   modelId: NumberLong(1),
//   chainId: NumberLong(8453),
//   network: 'base',
//   status: 'active',
//   createdAt: new Date(),
//   updatedAt: new Date()
// });

print('NFT contracts collection initialized');
"

echo -e "${GREEN}✓ MongoDB collections initialized${NC}"
echo ""

echo "======================================"
echo -e "${GREEN}Database Initialization Complete!${NC}"
echo "======================================"
