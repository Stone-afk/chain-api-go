package evm

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
)

type Address [20]byte
type Hash [32]byte
type Data []byte
type BlockRef struct{ wire string }

func Latest() BlockRef           { return BlockRef{wire: "latest"} }
func Pending() BlockRef          { return BlockRef{wire: "pending"} }
func AtNumber(n uint64) BlockRef { return BlockRef{wire: quantity(n)} }
func (b BlockRef) MarshalJSON() ([]byte, error) {
	if b.wire == "" {
		return nil, invalid()
	}
	return json.Marshal(b.wire)
}
func quantity(n uint64) string { return "0x" + strconv.FormatUint(n, 16) }
func ParseData(s string) (Data, error) {
	if !strings.HasPrefix(s, "0x") || len(s)%2 != 0 {
		return nil, invalid()
	}
	b, e := hex.DecodeString(s[2:])
	if e != nil {
		return nil, invalid()
	}
	return Data(b), nil
}
func ParseAddress(s string) (Address, error) {
	b, e := ParseData(s)
	if e != nil || len(b) != 20 {
		return Address{}, invalid()
	}
	return Address(b), nil
}
func ParseHash(s string) (Hash, error) {
	b, e := ParseData(s)
	if e != nil || len(b) != 32 {
		return Hash{}, invalid()
	}
	return Hash(b), nil
}
func (a Address) MarshalJSON() ([]byte, error) { return json.Marshal("0x" + hex.EncodeToString(a[:])) }
func (h Hash) MarshalJSON() ([]byte, error)    { return json.Marshal("0x" + hex.EncodeToString(h[:])) }
func (d Data) MarshalJSON() ([]byte, error)    { return json.Marshal("0x" + hex.EncodeToString(d)) }

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
