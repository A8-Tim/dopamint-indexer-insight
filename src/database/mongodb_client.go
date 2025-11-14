package database

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBConfig holds MongoDB connection configuration
type MongoDBConfig struct {
	URI            string
	Database       string
	Collection     string
	ConnectTimeout time.Duration
}

// DopamintMongoClient manages MongoDB connection for Dopamint data
type DopamintMongoClient struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
	config     MongoDBConfig
}

// NFTContractDocument represents the NFT contract document in MongoDB
type NFTContractDocument struct {
	ID              interface{} `bson:"_id,omitempty"`
	ContractAddress string      `bson:"contractAddress"`
	CollectionID    int64       `bson:"collectionId"`
	Creator         string      `bson:"creator"`
	Name            string      `bson:"name"`
	Symbol          string      `bson:"symbol"`
	BaseURI         string      `bson:"baseURI"`
	ModelID         int64       `bson:"modelId"`
	CreatedAt       time.Time   `bson:"createdAt"`
	UpdatedAt       time.Time   `bson:"updatedAt"`
	ChainID         int64       `bson:"chainId"`
	Network         string      `bson:"network"`
	Status          string      `bson:"status"` // active, inactive, etc.
}

// NewDopamintMongoClient creates a new MongoDB client
func NewDopamintMongoClient(config MongoDBConfig) (*DopamintMongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(config.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(config.Database)
	collection := database.Collection(config.Collection)

	fmt.Printf("[MongoDB] Connected to database: %s, collection: %s\n", config.Database, config.Collection)

	return &DopamintMongoClient{
		client:     client,
		database:   database,
		collection: collection,
		config:     config,
	}, nil
}

// GetNFTContractAddresses fetches all NFT contract addresses from MongoDB
func (m *DopamintMongoClient) GetNFTContractAddresses(ctx context.Context) ([]common.Address, error) {
	filter := bson.M{
		"status": bson.M{"$ne": "deleted"}, // Exclude deleted contracts
	}

	cursor, err := m.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	var addresses []common.Address
	for cursor.Next(ctx) {
		var doc NFTContractDocument
		if err := cursor.Decode(&doc); err != nil {
			fmt.Printf("[MongoDB] Failed to decode document: %v\n", err)
			continue
		}

		if doc.ContractAddress != "" && common.IsHexAddress(doc.ContractAddress) {
			addresses = append(addresses, common.HexToAddress(doc.ContractAddress))
		}
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return addresses, nil
}

// GetActiveNFTContracts fetches only active NFT contracts
func (m *DopamintMongoClient) GetActiveNFTContracts(ctx context.Context) ([]NFTContractDocument, error) {
	filter := bson.M{
		"status": "active",
	}

	cursor, err := m.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to query MongoDB: %w", err)
	}
	defer cursor.Close(ctx)

	var contracts []NFTContractDocument
	if err := cursor.All(ctx, &contracts); err != nil {
		return nil, fmt.Errorf("failed to decode contracts: %w", err)
	}

	return contracts, nil
}

// UpsertNFTContract inserts or updates an NFT contract
func (m *DopamintMongoClient) UpsertNFTContract(ctx context.Context, contract NFTContractDocument) error {
	contract.UpdatedAt = time.Now()
	if contract.CreatedAt.IsZero() {
		contract.CreatedAt = time.Now()
	}

	filter := bson.M{
		"contractAddress": contract.ContractAddress,
		"chainId":         contract.ChainID,
	}

	update := bson.M{
		"$set": contract,
		"$setOnInsert": bson.M{
			"createdAt": contract.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := m.collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert contract: %w", err)
	}

	if result.UpsertedCount > 0 {
		fmt.Printf("[MongoDB] Inserted new contract: %s\n", contract.ContractAddress)
	} else if result.ModifiedCount > 0 {
		fmt.Printf("[MongoDB] Updated contract: %s\n", contract.ContractAddress)
	}

	return nil
}

// GetContractByAddress fetches a contract by address
func (m *DopamintMongoClient) GetContractByAddress(ctx context.Context, address string, chainID int64) (*NFTContractDocument, error) {
	filter := bson.M{
		"contractAddress": address,
		"chainId":         chainID,
	}

	var contract NFTContractDocument
	err := m.collection.FindOne(ctx, filter).Decode(&contract)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch contract: %w", err)
	}

	return &contract, nil
}

// GetStats returns statistics about NFT contracts
func (m *DopamintMongoClient) GetStats(ctx context.Context) (map[string]interface{}, error) {
	totalCount, err := m.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count total documents: %w", err)
	}

	activeCount, err := m.collection.CountDocuments(ctx, bson.M{"status": "active"})
	if err != nil {
		return nil, fmt.Errorf("failed to count active documents: %w", err)
	}

	return map[string]interface{}{
		"total_contracts":  totalCount,
		"active_contracts": activeCount,
		"database":         m.config.Database,
		"collection":       m.config.Collection,
	}, nil
}

// Close closes the MongoDB connection
func (m *DopamintMongoClient) Close(ctx context.Context) error {
	if m.client != nil {
		return m.client.Disconnect(ctx)
	}
	return nil
}

// WatchNFTContracts watches for new NFT contract insertions (requires replica set)
func (m *DopamintMongoClient) WatchNFTContracts(ctx context.Context, callback func(contract NFTContractDocument)) error {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{
			{Key: "operationType", Value: bson.D{{Key: "$in", Value: bson.A{"insert", "update"}}}},
		}}},
	}

	stream, err := m.collection.Watch(ctx, pipeline)
	if err != nil {
		return fmt.Errorf("failed to create change stream: %w", err)
	}
	defer stream.Close(ctx)

	fmt.Println("[MongoDB] Watching for NFT contract changes...")

	for stream.Next(ctx) {
		var changeEvent struct {
			OperationType string              `bson:"operationType"`
			FullDocument  NFTContractDocument `bson:"fullDocument"`
		}

		if err := stream.Decode(&changeEvent); err != nil {
			fmt.Printf("[MongoDB] Failed to decode change event: %v\n", err)
			continue
		}

		callback(changeEvent.FullDocument)
	}

	if err := stream.Err(); err != nil {
		return fmt.Errorf("change stream error: %w", err)
	}

	return nil
}
