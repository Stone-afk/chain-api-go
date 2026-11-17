package evm

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
)

type Address string
type Hash string
type Data string

func (v Address) ToString() string { return string(v) }
func (v Hash) ToString() string    { return string(v) }
func (v Data) ToString() string    { return string(v) }

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
func validHex(s string, size int) bool {
	if !strings.HasPrefix(s, "0x") || len(s)%2 != 0 || (size >= 0 && len(s) != 2+size*2) {
		return false
	}
	_, err := hex.DecodeString(s[2:])
	return err == nil
}
func validQuantity(s string) bool {
	if len(s) < 3 || !strings.HasPrefix(s, "0x") || (len(s) > 3 && s[2] == '0') {
		return false
	}
	for _, c := range s[2:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}

type CallRequest struct {
	From                 string `json:"from,omitempty"`
	To                   string `json:"to,omitempty"`
	Gas                  string `json:"gas,omitempty"`
	Value                string `json:"value,omitempty"`
	GasPrice             string `json:"gasPrice,omitempty"`
	MaxFeePerGas         string `json:"maxFeePerGas,omitempty"`
	MaxPriorityFeePerGas string `json:"maxPriorityFeePerGas,omitempty"`
	Data                 string `json:"data,omitempty"`
}

type Block struct {
	Number        json.RawMessage `json:"number"`
	Hash          json.RawMessage `json:"hash"`
	BaseFeePerGas json.RawMessage `json:"baseFeePerGas"`
	ParentHash    string          `json:"parentHash"`
	Timestamp     string          `json:"timestamp"`
	GasLimit      string          `json:"gasLimit"`
	GasUsed       string          `json:"gasUsed"`
	Transactions  []string        `json:"transactions"`
}

type Transaction struct {
	Hash                 string          `json:"hash"`
	From                 string          `json:"from"`
	Nonce                string          `json:"nonce"`
	Gas                  string          `json:"gas"`
	Value                string          `json:"value"`
	Input                string          `json:"input"`
	To                   json.RawMessage `json:"to"`
	BlockHash            json.RawMessage `json:"blockHash"`
	BlockNumber          json.RawMessage `json:"blockNumber"`
	TransactionIndex     json.RawMessage `json:"transactionIndex"`
	GasPrice             json.RawMessage `json:"gasPrice"`
	MaxFeePerGas         json.RawMessage `json:"maxFeePerGas"`
	MaxPriorityFeePerGas json.RawMessage `json:"maxPriorityFeePerGas"`
	ChainID              json.RawMessage `json:"chainId"`
	Type                 json.RawMessage `json:"type"`
}

type Receipt struct {
	TransactionHash   string          `json:"transactionHash"`
	BlockHash         string          `json:"blockHash"`
	TransactionIndex  string          `json:"transactionIndex"`
	BlockNumber       string          `json:"blockNumber"`
	From              string          `json:"from"`
	CumulativeGasUsed string          `json:"cumulativeGasUsed"`
	GasUsed           string          `json:"gasUsed"`
	LogsBloom         string          `json:"logsBloom"`
	Logs              []Log           `json:"logs"`
	To                json.RawMessage `json:"to"`
	ContractAddress   json.RawMessage `json:"contractAddress"`
	Status            json.RawMessage `json:"status"`
	Root              json.RawMessage `json:"root"`
	EffectiveGasPrice json.RawMessage `json:"effectiveGasPrice"`
	Type              json.RawMessage `json:"type"`
}

type Log struct {
	Address          string          `json:"address"`
	Data             string          `json:"data"`
	Topics           []string        `json:"topics"`
	BlockHash        json.RawMessage `json:"blockHash"`
	TransactionHash  json.RawMessage `json:"transactionHash"`
	BlockNumber      json.RawMessage `json:"blockNumber"`
	TransactionIndex json.RawMessage `json:"transactionIndex"`
	LogIndex         json.RawMessage `json:"logIndex"`
	Removed          json.RawMessage `json:"removed"`
}

type FeeHistoryResult struct {
	OldestBlock   string          `json:"oldestBlock"`
	BaseFeePerGas []string        `json:"baseFeePerGas"`
	GasUsedRatio  []float64       `json:"gasUsedRatio"`
	Reward        json.RawMessage `json:"reward"`
}
type LogFilter struct {
	FromBlock, ToBlock BlockRef
	Addresses          []string
	Topics             [][]string
}
