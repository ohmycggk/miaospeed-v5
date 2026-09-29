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

func TestSignRequestV1EmptyRandomSequenceClearsNodesAndScripts(t *testing.T) {
	a := sampleRequest()
	a.RandomSequence = ""
	a.Configs.ApiVersion = interfaces.ApiV1
	a.Configs.Scripts = []interfaces.Script{{ID: "s", Content: "x"}}

	b := sampleRequest()
	b.RandomSequence = ""
	b.Configs.ApiVersion = interfaces.ApiV1
	b.Nodes = []interfaces.SlaveRequestNode{{Name: "other", Payload: "q"}}

	if SignRequest("token", a) != SignRequest("token", b) {
		t.Fatal("empty RandomSequence must ignore nodes and scripts in the signature")
	}

	legacy := legacyFrom(a)
	legacy.Configs.Scripts = make([]interfaces.Script, 0)
	legacy.Nodes = make([]interfaces.SlaveRequestNode, 0)
	if hashMiaoSpeed("token", marshalTrimmed(legacy)) != SignRequest("token", a) {
		t.Fatal("v1 empty sequence signature does not match the cleared payload")
	}
	if len(a.Nodes) != 1 || a.Nodes[0].Name != "n" || len(a.Configs.Scripts) != 1 {
		t.Fatal("signing must not clear nodes or scripts on the original request")
	}
}

func TestSignRequestV1NonEmptyRandomSequenceIncludesNodes(t *testing.T) {
	a := sampleRequest()
	a.Configs.ApiVersion = interfaces.ApiV1
	b := sampleRequest()
	b.Configs.ApiVersion = interfaces.ApiV1
	b.Nodes = []interfaces.SlaveRequestNode{{Name: "other", Payload: "q"}}
	if SignRequest("token", a) == SignRequest("token", b) {
		t.Fatal("non-empty RandomSequence must include nodes in the signature")
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
