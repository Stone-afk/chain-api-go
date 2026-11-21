package solana

import (
	srpc "github.com/solana-foundation/solana-go/v2/rpc"
	"reflect"
	"sort"
	"testing"
)

func TestNativeSolanaRestrictedSurface(t *testing.T) {
	c := client(t, &fakeCaller{})
	if _, ok := any(c).(nativeContract); !ok {
		t.Error("NS-01: native signatures missing")
	}
	typ := reflect.TypeOf(c)
	var names []string
	for i := 0; i < typ.NumMethod(); i++ {
		names = append(names, typ.Method(i).Name)
	}
	want := []string{"GetGenesisHash", "GetSlotWithOpts", "GetBlockWithOpts", "GetBlocks", "GetFirstAvailableBlock", "GetBalanceWithOpts", "GetAccountInfoWithOpts", "GetLatestBlockhashWithOpts", "GetSignatureStatuses", "GetTransaction", "GetFeeForMessageWithOpts", "SimulateRawTransactionWithOpts", "SendEncodedTransactionWithOpts"}
	sort.Strings(want)
	same(t, names, want)
	st := typ.Elem()
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		if (f.IsExported() || f.Anonymous) && f.Type == reflect.TypeOf((*srpc.Client)(nil)) {
			t.Fatal("full upstream client exposed")
		}
	}
}
