// Package chainapi documents a library for direct Ethereum, BNB, Bitcoin, Solana,
// and Tron RPC clients. Import the concrete chain package; this root package has
// no unified client, dispatcher, server, or runtime configuration loader.
// The module specifies Go 1.26.0 and fixes go-ethereum v1.17.10 and
// solana-go/v2 v2.3.0. EVM and Solana expose restricted native methods/types;
// their complete upstream clients stay private behind injected Caller transports.
// Old EVM/Solana DTOs and APIs were removed without a compatibility layer.
//
// All runtime dependencies are injected. Constructors perform no network I/O.
// The SDK does not construct, encode, or sign transactions, accept private keys,
// or invoke node signing/unlock APIs. Submission methods send caller-supplied
// serialized payloads once; acceptance does not establish execution or finality.
// Solana simulation accepts serialized bytes and only adds base64 RPC wrapping.
// There is no default network transport, retry, polling or background work.
// The fixed geth constructor reads crypto/rand entropy internally; zero network
// I/O does not imply absolute zero OS I/O.
//
// The current delivery has offline evidence only. Real browser, account RPC,
// and broadcast acceptance remains unexecuted and unauthorized in T036 (BLOCKED).
package chainapi
