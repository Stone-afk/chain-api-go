package evm

import "math/big"

type Address [20]byte
type Hash [32]byte
type Data []byte
type BlockRef struct{}

func (BlockRef) MarshalJSON() ([]byte, error) { return nil, unsupported() }

func ParseAddress(string) (Address, error)   { return Address{}, unsupported() }
func ParseHash(string) (Hash, error)         { return Hash{}, unsupported() }
func ParseData(string) (Data, error)         { return nil, unsupported() }
func (Address) MarshalJSON() ([]byte, error) { return nil, unsupported() }
func (Hash) MarshalJSON() ([]byte, error)    { return nil, unsupported() }
func (Data) MarshalJSON() ([]byte, error)    { return nil, unsupported() }
func Latest() BlockRef                       { return BlockRef{} }
func Pending() BlockRef                      { return BlockRef{} }
func AtNumber(uint64) BlockRef               { return BlockRef{} }

type CallRequest struct {
	From                    Address
	HasFrom                 bool
	To                      Address
	HasTo                   bool
	Gas                     uint64
	HasGas                  bool
	Value                   big.Int
	HasValue                bool
	GasPrice                big.Int
	HasGasPrice             bool
	MaxFeePerGas            big.Int
	HasMaxFeePerGas         bool
	MaxPriorityFeePerGas    big.Int
	HasMaxPriorityFeePerGas bool
	Data                    Data
}
type Block struct {
	Number                       uint64
	HasNumber                    bool
	Hash                         Hash
	HasHash                      bool
	ParentHash                   Hash
	Timestamp, GasLimit, GasUsed uint64
	Transactions                 []Hash
	BaseFeePerGas                big.Int
	HasBaseFeePerGas             bool
}
type Transaction struct {
	Hash                                                              Hash
	From                                                              Address
	Nonce, Gas                                                        uint64
	Value                                                             big.Int
	Input                                                             Data
	To                                                                Address
	HasTo                                                             bool
	BlockHash                                                         Hash
	HasBlockHash                                                      bool
	BlockNumber, TransactionIndex                                     uint64
	HasBlockNumber, HasTransactionIndex                               bool
	GasPrice, MaxFeePerGas, MaxPriorityFeePerGas, ChainID             big.Int
	HasGasPrice, HasMaxFeePerGas, HasMaxPriorityFeePerGas, HasChainID bool
	Type                                                              uint64
	HasType                                                           bool
}
type Receipt struct {
	TransactionHash, BlockHash    Hash
	TransactionIndex, BlockNumber uint64
	From                          Address
	To, ContractAddress           Address
	HasTo, HasContractAddress     bool
	CumulativeGasUsed, GasUsed    uint64
	Logs                          []Log
	LogsBloom                     [256]byte
	Status                        uint64
	HasStatus                     bool
	Root                          Hash
	HasRoot                       bool
	EffectiveGasPrice             big.Int
	HasEffectiveGasPrice          bool
	Type                          uint64
	HasType                       bool
}
type Log struct {
	Address                                          Address
	Topics                                           []Hash
	Data                                             Data
	Removed                                          bool
	BlockHash, TransactionHash                       Hash
	HasBlockHash, HasTransactionHash                 bool
	BlockNumber, TransactionIndex, LogIndex          uint64
	HasBlockNumber, HasTransactionIndex, HasLogIndex bool
}
type LogFilter struct {
	FromBlock, ToBlock BlockRef
	Addresses          []Address
	Topics             [][]Hash
}
type FeeHistoryResult struct {
	OldestBlock   uint64
	BaseFeePerGas []big.Int
	GasUsedRatio  []float64
	Reward        [][]big.Int
	HasReward     bool
}
