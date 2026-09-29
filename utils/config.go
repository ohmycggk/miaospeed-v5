package utils

import (
	"crypto/tls"
	"fmt"
	"os"
	"strings"

	"github.com/miaokobot/miaospeed/interfaces"
	"github.com/miaokobot/miaospeed/utils/structs"
)

type GlobalConfig struct {
	Token            string
	Binder           string
	WhiteList        []string
	SpeedLimit       uint64
	PauseSecond      uint
	ConnTaskTreading uint
	MiaoKoSignedTLS  bool
	NoSpeedFlag      bool
	MaxmindDB        string
	// Path is the required websocket path. Empty means every path is accepted.
	Path string
}

// NormalizeWSPath turns "koipy" and "/koipy" into "/koipy".
// An empty input stays empty so callers can keep the permissive default.
func NormalizeWSPath(raw string) string {
	if raw == "" {
		return ""
	}
	trimmed := strings.TrimLeft(raw, "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed
}

// ValidateWSPath reports whether an upgrade path is allowed.
// An unset configured path accepts every path.
func (gc *GlobalConfig) ValidateWSPath(path string) bool {
	if gc.Path == "" {
		return true
	}
	return path == gc.Path
}

// SplitWhiteList accepts both comma and pipe separators.
func SplitWhiteList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '|'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// ReadServerCertificatePair loads a PEM certificate and key.
// Both paths empty is a no-op. A missing, empty, or mismatched file is an error.
func ReadServerCertificatePair(pubPath, privPath string) (string, string, error) {
	if pubPath == "" && privPath == "" {
		return "", "", nil
	}
	if pubPath == "" || privPath == "" {
		return "", "", fmt.Errorf("both -serverpublickey and -serverprivatekey are required")
	}

	pub, err := os.ReadFile(pubPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read server public key %s: %w", pubPath, err)
	}
	key, err := os.ReadFile(privPath)
	if err != nil {
		return "", "", fmt.Errorf("cannot read server private key %s: %w", privPath, err)
	}
	if strings.TrimSpace(string(pub)) == "" || strings.TrimSpace(string(key)) == "" {
		return "", "", fmt.Errorf("server certificate or private key is empty")
	}
	if _, err := tls.X509KeyPair(pub, key); err != nil {
		return "", "", fmt.Errorf("server certificate and private key do not form a valid pair: %w", err)
	}
	return string(pub), string(key), nil
}

func (gc *GlobalConfig) InWhiteList(invoker string) bool {
	if len(gc.WhiteList) == 0 {
		return true
	}

	return structs.Contains(gc.WhiteList, invoker)
}

func (gc *GlobalConfig) VerifyRequest(req *interfaces.SlaveRequest) bool {
	return req.Challenge == gc.SignRequest(req)
}

func (gc *GlobalConfig) SignRequest(req *interfaces.SlaveRequest) string {
	return SignRequest(gc.Token, req)
}

var GCFG GlobalConfig
