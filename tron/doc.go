// Package tron exposes native FullNode and five explicit solidified HTTP APIs.
//
// New(httpjson.Poster) performs no validation or I/O; callers inject a valid Poster.
// Client queries are GetNodeInfo, GetNowBlock, GetBlockByNum, GetAccount,
// GetAccountResource, GetChainParameters, GetTransactionByID,
// GetTransactionInfoByID, GetSolidifiedAccount, GetSolidifiedNowBlock,
// GetSolidifiedBlockByNum, GetSolidifiedTransactionByID and
// GetSolidifiedTransactionInfoByID. TriggerConstantContract and EstimateEnergy
// compute at the node; BroadcastTransaction submits once. All take context first.
// RawPost remains on httpjson.Client; no additional solidified paths are inferred.
//
// Visible false uses 41-prefixed hex addresses; true uses T-prefixed base58.
// The SDK preserves the supplied encoding and does not calculate checksums.
// Balance, Fee and CallValue use sun; Energy and Bandwidth are separate resources.
// Proto3 may omit zero balance/resource/parameter values. An account with address
// and omitted balance exists with zero balance. Empty query objects mean absent;
// nonempty nullable results require address/blockID/txID/id as appropriate.
// Top-level null is invalid. Presence does not mean execution or confirmation.
//
// RawMessage retains absent/null/zero distinctions. HTTP 200 can contain native
// Error or result=false rejection. Original codes, messages, JSON and causes
// remain accessible and are not safe logs. Transaction ret and receipt/result
// are execution data, not transport errors.
//
// TransactionPayload carries the caller supplied raw JSON object. The SDK does not
// rebuild raw_data, signatures, TAPOS or txID. HTTP JSON marshaling can change
// insignificant whitespace/escaping. Broadcast success returns true and may omit
// txid; it proves acceptance only. SubmissionUnknown does not authorize replay.
// Solidified queries expose the node view without extra polling or confirmation.
package tron
