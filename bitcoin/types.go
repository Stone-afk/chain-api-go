package bitcoin

import (
	"encoding/json"
	"strings"
)

type Config struct{}

type TxID string
type BlockHash string
type RawTransaction string
type Script string
type EstimateMode string
type Satoshi int64
type SatoshiPerKVByte int64

func (v TxID) ToString() string           { return string(v) }
func (v BlockHash) ToString() string      { return string(v) }
func (v RawTransaction) ToString() string { return string(v) }
func (v Script) ToString() string         { return string(v) }
func (v EstimateMode) ToString() string   { return string(v) }
func (v Satoshi) ToInt64() int64          { return int64(v) }
func (v SatoshiPerKVByte) ToInt64() int64 { return int64(v) }

const (
	EstimateUnset        EstimateMode = "unset"
	EstimateEconomical   EstimateMode = "economical"
	EstimateConservative EstimateMode = "conservative"
)

type GetTxOutRequest struct {
	TxID              TxID
	Vout              uint32
	IncludeMempool    bool
	HasIncludeMempool bool
}
type GetRawTransactionRequest struct {
	TxID         TxID
	BlockHash    BlockHash
	HasBlockHash bool
}
type EstimateSmartFeeRequest struct {
	ConfTarget int64
	Mode       EstimateMode
	HasMode    bool
}

type BlockchainInfo struct {
	Chain         string          `json:"chain"`
	Blocks        int64           `json:"blocks"`
	Headers       int64           `json:"headers"`
	BestBlockHash string          `json:"bestblockhash"`
	Pruned        bool            `json:"pruned"`
	PruneHeight   json.RawMessage `json:"pruneheight"`
}

type Block struct {
	Hash              string          `json:"hash"`
	Height            int64           `json:"height"`
	Confirmations     int64           `json:"confirmations"`
	Time              int64           `json:"time"`
	TxIDs             []string        `json:"tx"`
	PreviousBlockHash json.RawMessage `json:"previousblockhash"`
	NextBlockHash     json.RawMessage `json:"nextblockhash"`
}

type ScriptPubKey struct {
	Hex     string          `json:"hex"`
	Type    string          `json:"type"`
	Address json.RawMessage `json:"address"`
}

type TxOut struct {
	BestBlock     string       `json:"bestblock"`
	Confirmations int64        `json:"confirmations"`
	Value         Satoshi      `json:"value"`
	ScriptPubKey  ScriptPubKey `json:"scriptPubKey"`
	Coinbase      bool         `json:"coinbase"`
}

type Transaction struct {
	TxID          string          `json:"txid"`
	WTxID         string          `json:"hash"`
	Hex           string          `json:"hex"`
	Version       int32           `json:"version"`
	LockTime      uint32          `json:"locktime"`
	Size          int64           `json:"size"`
	VSize         int64           `json:"vsize"`
	Weight        int64           `json:"weight"`
	Vin           []TxIn          `json:"vin"`
	Vout          []TxOutput      `json:"vout"`
	BlockHash     json.RawMessage `json:"blockhash"`
	Confirmations json.RawMessage `json:"confirmations"`
	BlockTime     json.RawMessage `json:"blocktime"`
	InActiveChain json.RawMessage `json:"in_active_chain"`
}

type TxIn struct {
	Sequence  uint32          `json:"sequence"`
	TxID      json.RawMessage `json:"txid"`
	Vout      json.RawMessage `json:"vout"`
	ScriptSig json.RawMessage `json:"scriptSig"`
	Coinbase  json.RawMessage `json:"coinbase"`
	Witness   json.RawMessage `json:"txinwitness"`
}

type TxOutput struct {
	N            uint32       `json:"n"`
	Value        Satoshi      `json:"value"`
	ScriptPubKey ScriptPubKey `json:"scriptPubKey"`
}

type MempoolEntry struct {
	VSize   int64    `json:"vsize"`
	Weight  int64    `json:"weight"`
	Time    int64    `json:"time"`
	Height  int64    `json:"height"`
	BaseFee Satoshi  `json:"base_fee"`
	Depends []string `json:"depends"`
	SpentBy []string `json:"spentby"`
}

type FeeEstimate struct {
	FeeRate SatoshiPerKVByte `json:"feerate"`
	Blocks  int64            `json:"blocks"`
	Errors  []string         `json:"errors"`
}

type MempoolAcceptResult struct {
	TxID          string          `json:"txid"`
	WTxID         string          `json:"wtxid"`
	Allowed       bool            `json:"allowed"`
	VSize         json.RawMessage `json:"vsize"`
	Fees          json.RawMessage `json:"fees"`
	RejectReason  string          `json:"reject-reason"`
	RejectDetails string          `json:"reject-details"`
}

func btcAmount(raw json.RawMessage) (int64, bool) {
	s := string(raw)
	if len(s) == 0 {
		return 0, false
	}
	negative := s[0] == '-'
	if negative {
		s = s[1:]
	}
	if len(s) == 0 || s[0] < '0' || s[0] > '9' {
		return 0, false
	}
	mantissa, exponent := s, ""
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		mantissa, exponent = s[:i], s[i+1:]
	}
	scale := 8
	if i := strings.IndexByte(mantissa, '.'); i >= 0 {
		scale -= len(mantissa) - i - 1
		mantissa = mantissa[:i] + mantissa[i+1:]
	}
	digits := strings.TrimLeft(mantissa, "0")
	if digits == "" {
		return 0, true
	}
	if negative {
		return 0, false
	}
	expNegative := false
	if len(exponent) > 0 && (exponent[0] == '+' || exponent[0] == '-') {
		expNegative = exponent[0] == '-'
		exponent = exponent[1:]
	}
	exp := 0
	for _, digit := range exponent {
		if exp > len(s)+19 {
			return 0, false
		}
		exp = exp*10 + int(digit-'0')
	}
	if expNegative {
		scale -= exp
	} else {
		scale += exp
	}
	if scale < 0 {
		remove := -scale
		if remove >= len(digits) || strings.TrimRight(digits[len(digits)-remove:], "0") != "" {
			return 0, false
		}
		digits = digits[:len(digits)-remove]
		scale = 0
	}
	if len(digits)+scale > 19 {
		return 0, false
	}
	var amount int64
	for _, digit := range digits {
		d := int64(digit - '0')
		if amount > (9223372036854775807-d)/10 {
			return 0, false
		}
		amount = amount*10 + d
	}
	for range scale {
		if amount > 9223372036854775807/10 {
			return 0, false
		}
		amount *= 10
	}
	return amount, true
}
