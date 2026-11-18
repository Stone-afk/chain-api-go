// Package solana exposes a restricted, directly imported Solana RPC client.
// It uses github.com/solana-foundation/solana-go/v2 v2.3.0 with Go 1.26.0.
// New(jsonrpc.Caller) privately creates the native client through
// NewWithCustomRPCClient. Its private Caller RPC client forwards CallForInto once to the
// injected Caller; callback and batch return Unsupported without calling out.
// Construction performs no network I/O, configuration reads or background work.
// No complete upstream client or default network transport is exposed.
//
// The eleven context-first methods are GetGenesisHash, GetSlotWithOpts,
// GetBlockWithOpts, GetBalanceWithOpts, GetAccountInfoWithOpts,
// GetLatestBlockhashWithOpts, GetSignatureStatuses, GetTransaction,
// GetFeeForMessageWithOpts, SimulateRawTransactionWithOpts and
// SendEncodedTransactionWithOpts. Parameters/results use native PublicKey, Hash,
// Signature and rpc types. GetSignatureStatuses takes a history bool followed
// by variadic signatures. Native options/results retain their pointer contracts.
// Old project DTOs, Has/present returns and old method names are removed without
// aliases or a compatibility layer. RawCall remains in the jsonrpc package.
//
// Native defaults and nullable values apply: block/account reads default to
// base64, null block returns rpc.ErrNotConfirmed, and null transaction or absent
// account returns rpc.ErrNotFound. Null status envelopes/Value return ErrNotFound;
// individual nil status entries retain order. Fee Value is *uint64: nil and a
// pointer to zero differ. Optional pointers preserve explicit zero/false.
// Legacy/v0, context slot, transaction meta and execution errors remain native.
// Balances/fees are lamports; slot is not LastValidBlockHeight.
//
// GetFeeForMessageWithOpts takes an already base64-encoded message string.
// SimulateRawTransactionWithOpts takes already serialized []byte, with upstream
// base64 transport wrapping only; it does not encode a transaction object.
// SendEncodedTransactionWithOpts takes an already encoded string and a value
// rpc.TransactionOpts, validates syntax, and submits the original payload once.
// Empty encoding defaults to base64; explicit base64/base58 are supported.
// SkipPreflight, PreflightCommitment, MaxRetries and MinContextSlot retain native
// semantics. Node MaxRetries is distinct from SDK retries, which are absent.
// A nonzero signature means acceptance, not execution or finality; uncertain
// outcomes are not safe to resend. Original Caller errors remain unwrap-accessible.
// Error() preserves raw text and is not a safe logging view.
//
// The SDK accepts no private keys and performs no transaction construction,
// transaction-object encoding, signing, node unlock, retries or polling.
package solana
