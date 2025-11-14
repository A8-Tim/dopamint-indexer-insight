package filters

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// EventListener listens for Factory events and auto-discovers new NFT contracts
type EventListener struct {
	contractFilter *ContractFilter
	factoryAddress common.Address
}

// Event signatures
var (
	// NFTContractCreated(uint256 indexed collectionId, address indexed contractAddress, address indexed creator, string name, string symbol, string baseURI)
	NFTContractCreatedSignature = crypto.Keccak256Hash([]byte("NFTContractCreated(uint256,address,address,string,string,string)"))
)

// NewEventListener creates a new event listener
func NewEventListener(contractFilter *ContractFilter, factoryAddress common.Address) *EventListener {
	return &EventListener{
		contractFilter: contractFilter,
		factoryAddress: factoryAddress,
	}
}

// ProcessLog processes a log entry and extracts NFT contract addresses
func (el *EventListener) ProcessLog(log types.Log) {
	// Only process logs from the factory contract
	if log.Address != el.factoryAddress {
		return
	}

	// Check if it's an NFTContractCreated event
	if len(log.Topics) == 0 || log.Topics[0] != NFTContractCreatedSignature {
		return
	}

	// Extract the contract address from the event
	// Topic layout:
	// Topic[0] = event signature
	// Topic[1] = contractAddress (indexed)
	// Topic[2] = creator (indexed)
	if len(log.Topics) < 3 {
		fmt.Printf("[EventListener] Invalid NFTContractCreated event: not enough topics\n")
		return
	}

	// Topic[1] contains the contract address (indexed address is 32 bytes with padding)
	contractAddress := common.BytesToAddress(log.Topics[1].Bytes())

	// Add to contract filter
	el.contractFilter.AddNFTContract(contractAddress)

	fmt.Printf("[EventListener] Discovered new NFT contract: %s\n", contractAddress.Hex())
}

// ProcessLogs processes multiple logs
func (el *EventListener) ProcessLogs(logs []types.Log) int {
	discoveredCount := 0
	for _, log := range logs {
		el.ProcessLog(log)
		// Check if it was an NFTContractCreated event
		if len(log.Topics) > 0 && log.Topics[0] == NFTContractCreatedSignature {
			discoveredCount++
		}
	}
	return discoveredCount
}

// NFTContractCreatedEvent represents the parsed NFTContractCreated event
type NFTContractCreatedEvent struct {
	CollectionID    *big.Int
	ContractAddress common.Address
	Creator         common.Address
	Name            string
	Symbol          string
	BaseURI         string
	BlockNumber     uint64
	TxHash          common.Hash
	LogIndex        uint
}

// ParseNFTContractCreatedEvent parses an NFTContractCreated event
func ParseNFTContractCreatedEvent(log types.Log) (*NFTContractCreatedEvent, error) {
	if len(log.Topics) < 3 {
		return nil, fmt.Errorf("invalid NFTContractCreated event: not enough topics")
	}

	if log.Topics[0] != NFTContractCreatedSignature {
		return nil, fmt.Errorf("not an NFTContractCreated event")
	}

	event := &NFTContractCreatedEvent{
		ContractAddress: common.BytesToAddress(log.Topics[1].Bytes()),
		Creator:         common.BytesToAddress(log.Topics[2].Bytes()),
		BlockNumber:     log.BlockNumber,
		TxHash:          log.TxHash,
		LogIndex:        log.Index,
	}

	// Parse the data field (collectionId, name, symbol, baseURI)
	// The data contains non-indexed parameters
	// For detailed parsing, we would need to use ABI decoder
	// For now, we just extract the contract address which is indexed

	return event, nil
}

// GetEventSignatures returns all event signatures to monitor
func GetEventSignatures() []common.Hash {
	return []common.Hash{
		NFTContractCreatedSignature,
	}
}

// BackfillNFTContracts backfills NFT contracts from historical events
func (el *EventListener) BackfillNFTContracts(ctx context.Context, logs []types.Log) error {
	fmt.Printf("[EventListener] Starting backfill of NFT contracts from %d logs\n", len(logs))

	discoveredCount := el.ProcessLogs(logs)

	fmt.Printf("[EventListener] Backfill complete: discovered %d NFT contracts\n", discoveredCount)

	return nil
}
