package utils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNormalizeWSPath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"koipy", "/koipy"},
		{"/koipy", "/koipy"},
		{"//koipy", "/koipy"},
		{"/", "/"},
	}
	for _, tc := range cases {
		if got := NormalizeWSPath(tc.in); got != tc.want {
			t.Errorf("NormalizeWSPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestValidateWSPath(t *testing.T) {
	open := &GlobalConfig{}
	if !open.ValidateWSPath("/anything") {
		t.Fatal("empty path should accept every upgrade path")
	}

	locked := &GlobalConfig{Path: "/koipy"}
	if !locked.ValidateWSPath("/koipy") {
		t.Fatal("matching path should be accepted")
	}
	if locked.ValidateWSPath("/") || locked.ValidateWSPath("/other") {
		t.Fatal("non-matching path should be rejected")
	}
}

func TestSplitWhiteList(t *testing.T) {
	if got := SplitWhiteList(""); len(got) != 0 {
		t.Fatalf("empty whitelist = %#v", got)
	}
	got := SplitWhiteList("1111,2222|3333")
	want := []string{"1111", "2222", "3333"}
	if len(got) != len(want) {
		t.Fatalf("SplitWhiteList = %#v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SplitWhiteList = %#v", got)
		}
	}
	if !(&GlobalConfig{WhiteList: got}).InWhiteList("2222") {
		t.Fatal("expected id from a pipe-separated list to be allowed")
	}
}

func TestReadServerCertificatePairMissing(t *testing.T) {
	if _, _, err := ReadServerCertificatePair("", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadServerCertificatePair("only.crt", ""); err == nil {
		t.Fatal("expected an error when only one path is set")
	}
	if _, _, err := ReadServerCertificatePair("/no/such.crt", "/no/such.key"); err == nil {
		t.Fatal("expected an error for a missing certificate")
	}

	dir := t.TempDir()
	pub := filepath.Join(dir, "empty.crt")
	key := filepath.Join(dir, "empty.key")
	if err := os.WriteFile(pub, []byte("  \n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("not-a-key"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ReadServerCertificatePair(pub, key); err == nil {
		t.Fatal("expected an error for an empty certificate")
	}
}

func TestReadServerCertificatePairValid(t *testing.T) {
	dir := t.TempDir()
	pubPath, keyPath := writeTestCert(t, dir)
	pub, key, err := ReadServerCertificatePair(pubPath, keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if pub == "" || key == "" {
		t.Fatal("expected PEM contents")
	}
}

func writeTestCert(t *testing.T, dir string) (string, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "miaospeed-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	pubPath := filepath.Join(dir, "server.crt")
	keyPath := filepath.Join(dir, "server.key")
	if err := os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatal(err)
	}
	return pubPath, keyPath
}
