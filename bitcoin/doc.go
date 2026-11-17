// Package bitcoin exposes a concrete Bitcoin Core client using jsonrpc.Caller.
//
// New(Config{}, caller) performs only local nil-interface validation. Use the
// Bitcoin protocol dialect when constructing a jsonrpc.Client. Client methods
// are GetBlockchainInfo, GetBlockCount, GetBlockHash, GetBlock, GetTxOut,
// GetRawTransaction, GetMempoolEntry, EstimateSmartFee, TestMempoolAccept and
// SendRawTransaction. All take context first; block and raw-transaction queries
// use verbosity 1. The protocol RawCall remains on jsonrpc.Client.
//
// GetTxOut null returns (TxOut{}, false, nil). EstimateSmartFee has a different
// bool: hasEstimate. Missing feerate preserves Blocks/Errors with false; explicit
// null is an error. Optional response RawMessage retains absent/null/zero.
// Optional request Has flags determine omission, including explicit false.
//
// Converted Value and BaseFee fields use Satoshi; FeeRate uses SatoshiPerKVByte.
// Both expose ToInt64. Conversion multiplies BTC by 10^8 exactly without floating
// point or rounding; fractional satoshis, negative amounts and overflow fail.
// 1000 sat/kvB is not 1000 sat/vB. MempoolAcceptResult.Fees remains raw node JSON.
// String wrappers expose ToString without conversion or byte reversal.
//
// Historical transactions require appropriate mempool, txindex or available block
// context. Errors are not replaced by empty results or another node. Preflight
// Allowed is not broadcast acceptance. Submission sends supplied hex once,
// without transaction construction or signing. Acceptance is not confirmation;
// SubmissionUnknown is not a retry instruction. Raw errors are not safe logs.
package bitcoin
