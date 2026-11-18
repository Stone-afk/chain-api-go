package tron

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/Stone-afk/chain-api-go/apierror"
)

func contractRequest() ContractCallRequest {
	return ContractCallRequest{OwnerAddress: hexAddress, ContractAddress: hexAddress, FunctionSelector: "balanceOf(address)", Parameter: ""}
}

type computeCase struct {
	name, path string
	call       func(*Client, context.Context, ContractCallRequest) (any, error)
	response   string
	want       any
}

func computeCases() []computeCase {
	return []computeCase{
		{"constant", "/wallet/triggerconstantcontract", func(c *Client, x context.Context, r ContractCallRequest) (any, error) {
			return c.TriggerConstantContract(x, r)
		}, `{"result":{"result":true,"code":"raw","message":"raw"},"constant_result":["00","ff"],"energy_used":17,"transaction":{"raw_data":{"n":9007199254740993}}}`, ConstantResult{Result: NodeResult{true, "raw", "raw"}, ConstantResult: []string{"00", "ff"}, EnergyUsed: 17, Transaction: json.RawMessage(`{"raw_data":{"n":9007199254740993}}`)}},
		{"energy", "/wallet/estimateenergy", func(c *Client, x context.Context, r ContractCallRequest) (any, error) { return c.EstimateEnergy(x, r) }, `{"result":{"result":true},"energy_required":23}`, EnergyEstimate{Result: NodeResult{Result: true}, EnergyRequired: 23}},
	}
}
func TestComputeWireAndResults(t *testing.T) {
	for _, method := range computeCases() {
		for _, tc := range []struct {
			name      string
			has       bool
			value     int64
			visible   bool
			parameter string
			body      string
		}{
			{"omitted", false, -5, false, "", `{"owner_address":"41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contract_address":"41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","function_selector":"balanceOf(address)","parameter":"","visible":false}`},
			{"zero", true, 0, false, "aB01", `{"owner_address":"41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","contract_address":"41aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","function_selector":"balanceOf(address)","parameter":"aB01","visible":false,"call_value":0}`},
			{"sun", true, 100, true, "", `{"owner_address":"T111111111111111111111111111111111","contract_address":"T111111111111111111111111111111111","function_selector":"balanceOf(address)","parameter":"","visible":true,"call_value":100}`},
		} {
			t.Run(method.name+tc.name, func(t *testing.T) {
				req := contractRequest()
				req.HasCallValue = tc.has
				req.CallValue = tc.value
				req.Visible = tc.visible
				req.Parameter = tc.parameter
				if tc.visible {
					req.OwnerAddress = base58Address
					req.ContractAddress = base58Address
				}
				p := &fakePoster{response: method.response}
				ctx := context.Background()
				got, err := method.call(newClient(t, p), ctx, req)
				if err != nil {
					t.Error("valid compute rejected")
				}
				if !reflect.DeepEqual(got, method.want) {
					t.Errorf("DTO mismatch: %#v", got)
				}
				checkWire(t, p, ctx, method.path, tc.body)
			})
		}
	}
}
func TestComputeValidation(t *testing.T) {
	for _, method := range computeCases() {
		for _, tc := range []struct {
			name   string
			change func(*ContractCallRequest)
		}{
			{"owner", func(r *ContractCallRequest) { r.OwnerAddress = "bad" }},
			{"contract", func(r *ContractCallRequest) { r.ContractAddress = "bad" }},
			{"visible", func(r *ContractCallRequest) { r.Visible = true }},
			{"selector", func(r *ContractCallRequest) { r.FunctionSelector = "" }},
			{"odd parameter", func(r *ContractCallRequest) { r.Parameter = "abc" }},
			{"prefixed parameter", func(r *ContractCallRequest) { r.Parameter = "0x00" }},
			{"nonhex parameter", func(r *ContractCallRequest) { r.Parameter = "zz" }},
			{"negative sun", func(r *ContractCallRequest) { r.HasCallValue = true; r.CallValue = -1 }},
		} {
			t.Run(method.name+tc.name, func(t *testing.T) {
				req := contractRequest()
				tc.change(&req)
				p := &fakePoster{}
				_, err := method.call(newClient(t, p), context.Background(), req)
				checkError(t, err, apierror.InvalidArgument, false)
				if p.calls != 0 {
					t.Error("invalid compute request sent")
				}
			})
		}
	}
}
func TestComputeResultBoundary(t *testing.T) {
	for _, method := range computeCases() {
		for _, response := range []string{`null`, `[]`, `{}`, `{"result":true}`, `{"result":null}`, `{"result":{}}`, `{"result":{"result":"false"}}`, `{"Error":null}`, `{"result":{"code":""}}`, `{"result":{"code":null,"message":"x"}}`, `{"result":{"code":7}}`, `{"result":{"message":"x"}}`, `{"result":{"result":null,"code":"OTHER_ERROR"}}`} {
			t.Run(method.name+response, func(t *testing.T) {
				p := &fakePoster{response: response}
				_, err := method.call(newClient(t, p), context.Background(), contractRequest())
				checkError(t, err, apierror.Protocol, false)
				if p.calls != 1 {
					t.Error("compute count mismatch")
				}
			})
		}
		for _, response := range []string{`{"result":{"result":false,"code":"CONTRACT_VALIDATE_ERROR","message":"  synthetic not found  "},"extra":9007199254740993}`, `{"result":{"code":"CONTRACT_VALIDATE_ERROR","message":"  synthetic not found  "},"extra":9007199254740993}`, `{"Error":"  synthetic unsupported  ","extra":9007199254740993}`} {
			t.Run(method.name+response, func(t *testing.T) {
				p := &fakePoster{response: response}
				_, err := method.call(newClient(t, p), context.Background(), contractRequest())
				e := checkError(t, err, apierror.RPC, false)
				if string(e.Data) != response {
					t.Error("raw response lost")
				}
				if response[2] == 'r' {
					if e.NodeCode != "CONTRACT_VALIDATE_ERROR" || e.Message != "  synthetic not found  " {
						t.Error("node strings changed")
					}
				} else {
					if e.Message != "  synthetic unsupported  " {
						t.Error("Error string changed")
					}
				}
				if e.HTTPStatus != 0 || len(e.ResponseBody) != 0 {
					t.Error("fabricated HTTP fields")
				}
				if e.Method != method.path || p.calls != 1 {
					t.Error("method/count mismatch")
				}
			})
		}
		t.Run(method.name+"proto3", func(t *testing.T) {
			p := &fakePoster{response: `{"result":{"result":true},"future":null}`}
			got, err := method.call(newClient(t, p), context.Background(), contractRequest())
			if err != nil {
				t.Error("proto3 omitted fields rejected")
			}
			if method.name == "constant" {
				if !reflect.DeepEqual(got, ConstantResult{Result: NodeResult{Result: true}}) {
					t.Error("constant defaults changed")
				}
			} else {
				if !reflect.DeepEqual(got, EnergyEstimate{Result: NodeResult{Result: true}}) {
					t.Error("energy defaults changed")
				}
			}
		})
	}
}
