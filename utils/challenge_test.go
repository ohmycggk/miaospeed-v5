package utils

import (
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"

	"github.com/miaokobot/miaospeed/interfaces"
)

func sampleRequest() *interfaces.SlaveRequest {
	return &interfaces.SlaveRequest{
		Basics: interfaces.SlaveRequestBasics{ID: "job", Invoker: "bot"},
		Options: interfaces.SlaveRequestOptions{
			Matrices: []interfaces.SlaveRequestMatrixEntry{{Type: interfaces.MatrixRTTPing}},
		},
		Configs: interfaces.SlaveRequestConfigs{
			PingAddress:    "http://gstatic.com/generate_204",
			TaskRetry:      3,
			UploadURL:      "https://speed.cloudflare.com/__up",
			UploadDuration: 8,
		},
		Vendor:         interfaces.VendorClash,
		Nodes:          []interfaces.SlaveRequestNode{{Name: "n", Payload: "p"}},
		RandomSequence: "abc",
		Challenge:      "ignore-me",
	}
}

func TestSignRequestV1IgnoresUploadFields(t *testing.T) {
	withUpload := sampleRequest()
	withoutUpload := sampleRequest()
	withoutUpload.Configs.UploadURL = ""
	withoutUpload.Configs.UploadDuration = 0
	withoutUpload.Configs.ApiVersion = 0

	if SignRequest("token", withUpload) != SignRequest("token", withoutUpload) {
		t.Fatal("api v0/v1 signature must ignore upload fields")
	}

	legacy := legacyFrom(withUpload)
	raw := marshalTrimmed(legacy)
	if strings.Contains(raw, "UploadURL") || strings.Contains(raw, "ApiVersion") {
		t.Fatalf("legacy signature payload includes new fields: %s", raw)
	}
	if hashMiaoSpeed("token", raw) != SignRequest("token", withUpload) {
		t.Fatal("v1 signature does not match the legacy payload")
	}
}

func TestSignRequestV3IncludesUploadFields(t *testing.T) {
	req := sampleRequest()
	req.Configs.ApiVersion = interfaces.ApiV3
	signed := req.Clone()
	signed.Challenge = ""
	raw, err := jsoniter.MarshalToString(signed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, `"UploadURL":"https://speed.cloudflare.com/__up"`) {
		t.Fatalf("v3 payload missing upload url: %s", raw)
	}
	if !strings.Contains(raw, `"ApiVersion":3`) {
		t.Fatalf("v3 payload missing api version: %s", raw)
	}
	if hashMiaoSpeed("token", strings.TrimSpace(raw)) != SignRequest("token", req) {
		t.Fatal("v3 signature does not match the full request payload")
	}

	other := sampleRequest()
	other.Configs.ApiVersion = interfaces.ApiV3
	other.Configs.UploadURL = "https://example.invalid/up"
	if SignRequest("token", req) == SignRequest("token", other) {
		t.Fatal("different upload urls must change the v3 signature")
	}
}

func TestSignRequestRejectsUnknownAPIVersion(t *testing.T) {
	req := sampleRequest()
	req.Configs.ApiVersion = 9
	first := SignRequest("token", req)
	second := SignRequest("token", req)
	if first == "" || first == second {
		t.Fatal("unknown api versions must not produce a stable signature")
	}
}
