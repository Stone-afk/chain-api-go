package bitcoin

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestScalarTypes(t *testing.T) {
	for _, tc := range []struct {
		value     any
		kind      reflect.Kind
		converted any
		want      any
	}{
		{TxID("Ab"), reflect.String, TxID("Ab").ToString(), "Ab"},
		{BlockHash("Ab"), reflect.String, BlockHash("Ab").ToString(), "Ab"},
		{RawTransaction("Ab"), reflect.String, RawTransaction("Ab").ToString(), "Ab"},
		{Script("Ab"), reflect.String, Script("Ab").ToString(), "Ab"},
		{EstimateMode("unset"), reflect.String, EstimateMode("unset").ToString(), "unset"},
		{Satoshi(-1), reflect.Int64, Satoshi(-1).ToInt64(), int64(-1)},
		{SatoshiPerKVByte(1000), reflect.Int64, SatoshiPerKVByte(1000).ToInt64(), int64(1000)},
	} {
		equal(t, reflect.TypeOf(tc.value).Kind(), tc.kind)
		equal(t, tc.converted, tc.want)
	}
	equal(t, EstimateUnset, EstimateMode("unset"))
	equal(t, EstimateEconomical, EstimateMode("economical"))
	equal(t, EstimateConservative, EstimateMode("conservative"))
}
func TestDTOFields(t *testing.T) {
	type field struct {
		name string
		typ  reflect.Type
		tag  string
	}
	for _, tc := range []struct {
		value  any
		fields []field
	}{
		{BlockchainInfo{}, []field{
			{"Chain", reflect.TypeOf(""), "chain"},
			{"Blocks", reflect.TypeOf(int64(0)), "blocks"},
			{"Headers", reflect.TypeOf(int64(0)), "headers"},
			{"BestBlockHash", reflect.TypeOf(""), "bestblockhash"},
			{"Pruned", reflect.TypeOf(false), "pruned"},
			{"PruneHeight", reflect.TypeOf(json.RawMessage{}), "pruneheight"},
		}},
		{Block{}, []field{
			{"Hash", reflect.TypeOf(""), "hash"},
			{"Height", reflect.TypeOf(int64(0)), "height"},
			{"Confirmations", reflect.TypeOf(int64(0)), "confirmations"},
			{"Time", reflect.TypeOf(int64(0)), "time"},
			{"TxIDs", reflect.TypeOf([]string{}), "tx"},
			{"PreviousBlockHash", reflect.TypeOf(json.RawMessage{}), "previousblockhash"},
			{"NextBlockHash", reflect.TypeOf(json.RawMessage{}), "nextblockhash"},
		}},
		{ScriptPubKey{}, []field{
			{"Hex", reflect.TypeOf(""), "hex"},
			{"Type", reflect.TypeOf(""), "type"},
			{"Address", reflect.TypeOf(json.RawMessage{}), "address"},
		}},
		{TxOut{}, []field{
			{"BestBlock", reflect.TypeOf(""), "bestblock"},
			{"Confirmations", reflect.TypeOf(int64(0)), "confirmations"},
			{"Value", reflect.TypeOf(Satoshi(0)), "value"},
			{"ScriptPubKey", reflect.TypeOf(ScriptPubKey{}), "scriptPubKey"},
			{"Coinbase", reflect.TypeOf(false), "coinbase"},
		}},
		{Transaction{}, []field{
			{"TxID", reflect.TypeOf(""), "txid"},
			{"WTxID", reflect.TypeOf(""), "hash"},
			{"Hex", reflect.TypeOf(""), "hex"},
			{"Version", reflect.TypeOf(int32(0)), "version"},
			{"LockTime", reflect.TypeOf(uint32(0)), "locktime"},
			{"Size", reflect.TypeOf(int64(0)), "size"},
			{"VSize", reflect.TypeOf(int64(0)), "vsize"},
			{"Weight", reflect.TypeOf(int64(0)), "weight"},
			{"Vin", reflect.TypeOf([]TxIn{}), "vin"},
			{"Vout", reflect.TypeOf([]TxOutput{}), "vout"},
			{"BlockHash", reflect.TypeOf(json.RawMessage{}), "blockhash"},
			{"Confirmations", reflect.TypeOf(json.RawMessage{}), "confirmations"},
			{"BlockTime", reflect.TypeOf(json.RawMessage{}), "blocktime"},
			{"InActiveChain", reflect.TypeOf(json.RawMessage{}), "in_active_chain"},
		}},
		{TxIn{}, []field{
			{"Sequence", reflect.TypeOf(uint32(0)), "sequence"},
			{"TxID", reflect.TypeOf(json.RawMessage{}), "txid"},
			{"Vout", reflect.TypeOf(json.RawMessage{}), "vout"},
			{"ScriptSig", reflect.TypeOf(json.RawMessage{}), "scriptSig"},
			{"Coinbase", reflect.TypeOf(json.RawMessage{}), "coinbase"},
			{"Witness", reflect.TypeOf(json.RawMessage{}), "txinwitness"},
		}},
		{TxOutput{}, []field{
			{"N", reflect.TypeOf(uint32(0)), "n"},
			{"Value", reflect.TypeOf(Satoshi(0)), "value"},
			{"ScriptPubKey", reflect.TypeOf(ScriptPubKey{}), "scriptPubKey"},
		}},
		{MempoolEntry{}, []field{
			{"VSize", reflect.TypeOf(int64(0)), "vsize"},
			{"Weight", reflect.TypeOf(int64(0)), "weight"},
			{"Time", reflect.TypeOf(int64(0)), "time"},
			{"Height", reflect.TypeOf(int64(0)), "height"},
			{"BaseFee", reflect.TypeOf(Satoshi(0)), "base_fee"},
			{"Depends", reflect.TypeOf([]string{}), "depends"},
			{"SpentBy", reflect.TypeOf([]string{}), "spentby"},
		}},
		{FeeEstimate{}, []field{
			{"FeeRate", reflect.TypeOf(SatoshiPerKVByte(0)), "feerate"},
			{"Blocks", reflect.TypeOf(int64(0)), "blocks"},
			{"Errors", reflect.TypeOf([]string{}), "errors"},
		}},
		{MempoolAcceptResult{}, []field{
			{"TxID", reflect.TypeOf(""), "txid"},
			{"WTxID", reflect.TypeOf(""), "wtxid"},
			{"Allowed", reflect.TypeOf(false), "allowed"},
			{"VSize", reflect.TypeOf(json.RawMessage{}), "vsize"},
			{"Fees", reflect.TypeOf(json.RawMessage{}), "fees"},
			{"RejectReason", reflect.TypeOf(""), "reject-reason"},
			{"RejectDetails", reflect.TypeOf(""), "reject-details"},
		}},
	} {
		t.Run(reflect.TypeOf(tc.value).Name(), func(t *testing.T) {
			typ := reflect.TypeOf(tc.value)
			equal(t, typ.NumField(), len(tc.fields))
			for _, want := range tc.fields {
				got, ok := typ.FieldByName(want.name)
				if !ok {
					t.Errorf("missing %s", want.name)
					continue
				}
				equal(t, got.Type, want.typ)
				equal(t, got.Tag.Get("json"), want.tag)
			}
		})
	}
}
func TestOptionalRawStates(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{`{}`, ""}, {`{"blockhash":null}`, "null"}, {`{"blockhash":0}`, "0"}, {`{"blockhash":"native"}`, `"native"`}} {
		t.Run(tc.input, func(t *testing.T) {
			var v Transaction
			e := json.Unmarshal([]byte(tc.input), &v)
			equal(t, e, nil)
			equal(t, string(v.BlockHash), tc.want)
		})
	}
	var v Block
	e := json.Unmarshal([]byte(`{"height":1,"height":2,"future":{"field":true},"tx":["unvalidated"]}`), &v)
	equal(t, e, nil)
	equal(t, v.Height, int64(2))
	equal(t, v.TxIDs, []string{"unvalidated"})
}
