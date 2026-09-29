package utils

import (
	"crypto/md5"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/miaokobot/miaospeed/interfaces"
)

// aws-v4-signature-like signing method
func hashMiaoSpeed(token, request string) string {
	buildTokens := append([]string{token}, strings.Split(strings.TrimSpace(BUILDTOKEN), "|")...)

	hasher := sha512.New()
	hasher.Write([]byte(request))

	for _, t := range buildTokens {
		if t == "" {
			// unsafe, plase make sure not to let token segment be empty
			t = "SOME_TOKEN"
		}

		hasher.Write(hasher.Sum([]byte(t)))
	}

	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

func hashMd5(token string) string {
	hasher := md5.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}

// legacyConfigs is the request config shape signed by clients that do not send ApiVersion.
type legacyConfigs struct {
	STUNURL           string
	DownloadURL       string
	DownloadDuration  int64
	DownloadThreading uint
	PingAverageOver   uint16
	PingAddress       string
	TaskRetry         uint
	DNSServers        []string
	TaskTimeout       uint
	Scripts           []interfaces.Script
}

type legacyRequest struct {
	Basics         interfaces.SlaveRequestBasics
	Options        interfaces.SlaveRequestOptions
	Configs        legacyConfigs
	Vendor         interfaces.VendorType
	Nodes          []interfaces.SlaveRequestNode
	RandomSequence string
	Challenge      string
}

// v2Configs adds ApiVersion without the upload fields introduced in API v3.
type v2Configs struct {
	legacyConfigs
	ApiVersion int
}

type v2Request struct {
	Basics         interfaces.SlaveRequestBasics
	Options        interfaces.SlaveRequestOptions
	Configs        v2Configs
	Vendor         interfaces.VendorType
	Nodes          []interfaces.SlaveRequestNode
	RandomSequence string
	Challenge      string
}

func legacyFrom(req *interfaces.SlaveRequest) legacyRequest {
	cloned := req.Clone()
	cloned.Challenge = ""
	return legacyRequest{
		Basics:  cloned.Basics,
		Options: cloned.Options,
		Configs: legacyConfigs{
			STUNURL:           cloned.Configs.STUNURL,
			DownloadURL:       cloned.Configs.DownloadURL,
			DownloadDuration:  cloned.Configs.DownloadDuration,
			DownloadThreading: cloned.Configs.DownloadThreading,
			PingAverageOver:   cloned.Configs.PingAverageOver,
			PingAddress:       cloned.Configs.PingAddress,
			TaskRetry:         cloned.Configs.TaskRetry,
			DNSServers:        cloned.Configs.DNSServers,
			TaskTimeout:       cloned.Configs.TaskTimeout,
			Scripts:           cloned.Configs.Scripts,
		},
		Vendor:         cloned.Vendor,
		Nodes:          cloned.Nodes,
		RandomSequence: cloned.RandomSequence,
	}
}

func marshalTrimmed(v any) string {
	raw, _ := jsoniter.MarshalToString(v)
	return strings.TrimSpace(raw)
}

func SignRequest(token string, req *interfaces.SlaveRequest) string {
	if req == nil {
		return hashMiaoSpeed(token, "")
	}

	switch req.Configs.ApiVersion {
	case interfaces.ApiV0, interfaces.ApiV1:
		legacy := legacyFrom(req)
		if req.RandomSequence == "" {
			// fulltclash Premium compatibility; fulltclash is being deprecated.
			DWarn("MiaoServer compatibility deprecation: this change will be deprecated in future versions. Please upgrade your client version.")
			legacy.Configs.Scripts = make([]interfaces.Script, 0)
			legacy.Nodes = make([]interfaces.SlaveRequestNode, 0)
		}
		return hashMiaoSpeed(token, marshalTrimmed(legacy))
	case interfaces.ApiV2:
		legacy := legacyFrom(req)
		return hashMiaoSpeed(token, marshalTrimmed(v2Request{
			Basics:  legacy.Basics,
			Options: legacy.Options,
			Configs: v2Configs{
				legacyConfigs: legacy.Configs,
				ApiVersion:    interfaces.ApiV2,
			},
			Vendor:         legacy.Vendor,
			Nodes:          legacy.Nodes,
			RandomSequence: legacy.RandomSequence,
		}))
	case interfaces.ApiV3:
		awaitSigned := req.Clone()
		awaitSigned.Challenge = ""
		return hashMiaoSpeed(token, marshalTrimmed(awaitSigned))
	default:
		return RandomUUID()
	}
}
