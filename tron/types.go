package tron

import (
	"encoding/json"
	"strings"
)

type Config struct{}
type Address string
type TxID string

func (a Address) ToString() string { return string(a) }
func (id TxID) ToString() string   { return string(id) }

type TransactionPayload struct{ JSON json.RawMessage }
type GetAccountRequest struct {
	Address Address
	Visible bool
}
type ContractCallRequest struct {
	OwnerAddress, ContractAddress Address
	FunctionSelector, Parameter   string
	CallValue                     int64
	HasCallValue                  bool
	Visible                       bool
}
type NodeInfo struct {
	Block               string          `json:"block"`
	SolidityBlock       string          `json:"solidityBlock"`
	CurrentConnectCount int64           `json:"currentConnectCount"`
	ConfigNodeInfo      json.RawMessage `json:"configNodeInfo"`
	PeerList            json.RawMessage `json:"peerList"`
}
type Block struct {
	BlockID      string          `json:"blockID"`
	BlockHeader  json.RawMessage `json:"block_header"`
	Transactions []Transaction   `json:"transactions"`
}
type Account struct {
	Address         string          `json:"address"`
	Balance         int64           `json:"balance"`
	AccountResource json.RawMessage `json:"account_resource"`
	AssetV2         json.RawMessage `json:"assetV2"`
}
type AccountResource struct {
	FreeNetUsed       int64 `json:"freeNetUsed"`
	FreeNetLimit      int64 `json:"freeNetLimit"`
	NetUsed           int64 `json:"NetUsed"`
	NetLimit          int64 `json:"NetLimit"`
	EnergyUsed        int64 `json:"EnergyUsed"`
	EnergyLimit       int64 `json:"EnergyLimit"`
	TotalNetLimit     int64 `json:"TotalNetLimit"`
	TotalNetWeight    int64 `json:"TotalNetWeight"`
	TotalEnergyLimit  int64 `json:"TotalEnergyLimit"`
	TotalEnergyWeight int64 `json:"TotalEnergyWeight"`
}
type ChainParameter struct {
	Key   string `json:"key"`
	Value int64  `json:"value"`
}
type ChainParameters struct {
	ChainParameter []ChainParameter `json:"chainParameter"`
}
type Transaction struct {
	TxID       string          `json:"txID"`
	RawData    json.RawMessage `json:"raw_data"`
	RawDataHex string          `json:"raw_data_hex"`
	Signature  []string        `json:"signature"`
	Ret        json.RawMessage `json:"ret"`
}
type TransactionInfo struct {
	ID                   string          `json:"id"`
	Fee                  int64           `json:"fee"`
	BlockNumber          int64           `json:"blockNumber"`
	BlockTimeStamp       int64           `json:"blockTimeStamp"`
	ContractResult       []string        `json:"contractResult"`
	ContractAddress      string          `json:"contract_address"`
	Receipt              json.RawMessage `json:"receipt"`
	Log                  json.RawMessage `json:"log"`
	InternalTransactions json.RawMessage `json:"internal_transactions"`
	Result               json.RawMessage `json:"result"`
	ResMessage           json.RawMessage `json:"resMessage"`
}
type NodeResult struct {
	Result  bool   `json:"result"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type ConstantResult struct {
	Result         NodeResult      `json:"result"`
	ConstantResult []string        `json:"constant_result"`
	EnergyUsed     int64           `json:"energy_used"`
	Transaction    json.RawMessage `json:"transaction"`
}
type EnergyEstimate struct {
	Result         NodeResult `json:"result"`
	EnergyRequired int64      `json:"energy_required"`
}
type BroadcastResult struct {
	Result  bool   `json:"result"`
	TxID    string `json:"txid"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func validHex(s string) bool {
	for _, ch := range s {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F') {
			return false
		}
	}
	return true
}
func validTxID(s string) bool { return len(s) == 64 && validHex(s) }
func validAddress(address Address, visible bool) bool {
	s := string(address)
	if !visible {
		return len(s) == 42 && strings.HasPrefix(s, "41") && validHex(s)
	}
	if len(s) != 34 || s[0] != 'T' {
		return false
	}
	for _, ch := range s {
		if !strings.ContainsRune("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", ch) {
			return false
		}
	}
	return true
}
