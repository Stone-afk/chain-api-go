// Package evm provides the restricted native EVM API shared by Ethereum and BNB.
// It requires Go 1.26.0 and github.com/ethereum/go-ethereum v1.17.10.
//
// New requires an injected jsonrpc.Caller and rejects a nil interface. It makes
// no network or Caller requests. A private in-memory HTTP bridge connects the
// private upstream clients to Caller, with no default network transport. The
// fixed upstream constructor reads crypto/rand entropy; zero network I/O does
// not mean zero OS I/O. No complete upstream client or lifecycle accessor is
// exposed. The host owns the actual transport, endpoint, headers and deadlines.
//
// The thirteen query/compute methods delegate to ethclient: ChainID, BlockNumber,
// BlockByNumber, BalanceAt, NonceAt, CodeAt, TransactionByHash,
// TransactionReceipt, FilterLogs, CallContract, EstimateGas, SuggestGasPrice and
// FeeHistory. They use common.Address/Hash, core/types.Block/Transaction/Receipt/
// Log, ethereum.CallMsg/FilterQuery/FeeHistory, *big.Int and byte slices directly.
// BalanceAt returns wei, SuggestGasPrice wei/gas and EstimateGas gas units.
//
// Old project DTOs, BlockRef helpers and Get* query APIs were removed without a
// compatibility layer. Native pointer, null, default and block-tag semantics
// apply: nil block numbers mean latest; null blocks, transactions and receipts
// return ethereum.NotFound. TransactionByHash's bool means isPending, not present.
// BlockByNumber returns full transactions and may perform one block request plus
// ordered uncle requests; the bridge stops on the first Caller error. Integration
// material version 2 budgets each underlying Caller request, including uncles.
//
// SendRawTransaction accepts only an already encoded string with a lower-case 0x
// prefix and nonempty even hex, sends it unchanged once and returns common.Hash.
// Construction, signing and transaction encoding belong outside this SDK. It
// offers no private-key, transaction-object construction, encoding or signing API.
// A nonzero hash means node acceptance only, not execution or confirmation.
// Local invalid input makes no request. RPC rejection is known; uncertain
// transport/protocol outcomes use apierror.Error.SubmissionUnknown.
//
// Caller errors remain reachable with errors.Is/As through native/HTTP wrappers;
// raw node fields, response bodies and Cause are not re-encoded as error envelopes.
// Native errors need not be apierror.Error. Returned errors are not safe log text.
// There are no retries, background tasks, polling, caching or node fallback.
// T036 real RPC and broadcast acceptance remains BLOCKED and has not been run.
package evm
