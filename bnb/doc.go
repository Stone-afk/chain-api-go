// Package bnb provides a concrete BNB Chain client using the restricted
// evm.Client API, Go 1.26.0 and go-ethereum v1.17.10.
//
// New(Config, jsonrpc.Caller) requires a positive ExpectedChainID big.Int value,
// copies it, and performs no network or Caller requests. ValidateNetwork(ctx)
// explicitly calls ChainID once and reports NetworkMismatch when different.
// Validation is not automatic and does not guarantee later provider identity.
//
// Client embeds only the project's restricted *evm.Client. Its thirteen native
// query/compute methods are ChainID, BlockNumber, BlockByNumber, BalanceAt,
// NonceAt, CodeAt, TransactionByHash, TransactionReceipt, FilterLogs,
// CallContract, EstimateGas, SuggestGasPrice and FeeHistory. SendRawTransaction
// accepts an externally constructed, signed and encoded string, sent once.
// A hash means node acceptance only. See package evm for native types, pointer,
// null/NotFound, pending/default semantics and breaking removal of old DTO/APIs.
//
// The private in-memory bridge has no default network transport and exposes no
// complete upstream client. The host owns dependencies and actual transport
// lifecycle. There are no retries, background tasks or node fallback. Errors
// preserve raw Caller fields and Cause through errors.Is/As; they are not safe
// log text. The upstream constructor's entropy read is not network I/O.
package bnb
