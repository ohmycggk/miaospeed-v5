package tests

import (
	"testing"

	"github.com/miaokobot/miaospeed/interfaces"
	clashvendor "github.com/miaokobot/miaospeed/vendors/clash"
)

func buildClashVendor(proxyName, proxyPayload string) interfaces.Vendor {
	return (&clashvendor.Clash{}).Build(proxyName, proxyPayload)
}

func TestClashBuild_InvalidYAMLNotReady(t *testing.T) {
	vendor := buildClashVendor("invalid", "name: [")
	if vendor.Status() != interfaces.VStatusNotReady {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusNotReady, vendor.Status())
	}
}

func TestClashBuild_VlessXHTTPReuseSettings(t *testing.T) {
	vendor := buildClashVendor("vless-xhttp", `
name: vless-xhttp
type: vless
server: example.com
port: 443
uuid: 11111111-1111-1111-1111-111111111111
udp: true
tls: true
network: xhttp
servername: example.com
client-fingerprint: chrome
xhttp-opts:
  path: /
  host: example.com
  mode: stream-up
  reuse-settings:
    max-connections: "16-32"
    max-concurrency: "0"
    c-max-reuse-times: "0"
    h-max-request-times: "600-900"
    h-max-reusable-secs: "1800-3000"
`)

	if vendor.Status() != interfaces.VStatusOperational {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusOperational, vendor.Status())
	}

	proxyInfo := vendor.ProxyInfo()
	if proxyInfo.Name != "vless-xhttp" {
		t.Fatalf("expected proxy name vless-xhttp, got %s", proxyInfo.Name)
	}
	if proxyInfo.Type != interfaces.Vless {
		t.Fatalf("expected proxy type %s, got %s", interfaces.Vless, proxyInfo.Type)
	}
}

func TestClashBuild_VmessGrpcExtendedOptions(t *testing.T) {
	vendor := buildClashVendor("vmess-grpc", `
name: vmess-grpc
type: vmess
server: example.com
port: 443
uuid: 11111111-1111-1111-1111-111111111111
alterId: 32
cipher: auto
network: grpc
tls: true
servername: example.com
grpc-opts:
  grpc-service-name: example
  ping-interval: 10
  max-connections: 2
  min-streams: 1
`)

	if vendor.Status() != interfaces.VStatusOperational {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusOperational, vendor.Status())
	}

	proxyInfo := vendor.ProxyInfo()
	if proxyInfo.Name != "vmess-grpc" {
		t.Fatalf("expected proxy name vmess-grpc, got %s", proxyInfo.Name)
	}
	if proxyInfo.Type != interfaces.Vmess {
		t.Fatalf("expected proxy type %s, got %s", interfaces.Vmess, proxyInfo.Type)
	}
}

func TestClashBuild_TrustTunnel(t *testing.T) {
	vendor := buildClashVendor("trusttunnel", `
name: trusttunnel
type: trusttunnel
server: 1.2.3.4
port: 443
username: username
password: password
health-check: true
udp: true
`)

	if vendor.Status() != interfaces.VStatusOperational {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusOperational, vendor.Status())
	}

	proxyInfo := vendor.ProxyInfo()
	if proxyInfo.Name != "trusttunnel" {
		t.Fatalf("expected proxy name trusttunnel, got %s", proxyInfo.Name)
	}
	if proxyInfo.Type != interfaces.TrustTunnel {
		t.Fatalf("expected proxy type %s, got %s", interfaces.TrustTunnel, proxyInfo.Type)
	}
}

func TestClashBuild_Nowhere3TCP(t *testing.T) {
	vendor := buildClashVendor("nowhere3-tcp", `
name: nowhere3-tcp
type: nowhere
server: 1.2.3.4
port: 443
password: 0123456789abcdef0123456789abcdef
udp: true
`)

	if vendor.Status() != interfaces.VStatusOperational {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusOperational, vendor.Status())
	}

	proxyInfo := vendor.ProxyInfo()
	if proxyInfo.Name != "nowhere3-tcp" {
		t.Fatalf("expected proxy name nowhere3-tcp, got %s", proxyInfo.Name)
	}
	if proxyInfo.Type != interfaces.Nowhere3 {
		t.Fatalf("expected proxy type %s, got %s", interfaces.Nowhere3, proxyInfo.Type)
	}
}

func TestClashBuild_Nowhere3QUIC(t *testing.T) {
	vendor := buildClashVendor("nowhere3-quic", `
name: nowhere3-quic
type: nowhere
server: 1.2.3.4
port: 443
password: 0123456789abcdef0123456789abcdef
net: udp
congestion-controller: bbr
cwnd: 32
udp: true
`)

	if vendor.Status() != interfaces.VStatusOperational {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusOperational, vendor.Status())
	}

	proxyInfo := vendor.ProxyInfo()
	if proxyInfo.Name != "nowhere3-quic" {
		t.Fatalf("expected proxy name nowhere3-quic, got %s", proxyInfo.Name)
	}
	if proxyInfo.Type != interfaces.Nowhere3 {
		t.Fatalf("expected proxy type %s, got %s", interfaces.Nowhere3, proxyInfo.Type)
	}
}

func TestClashBuild_Nowhere2Legacy(t *testing.T) {
	vendor := buildClashVendor("nowhere2", `
name: nowhere2
type: nowhere
server: 1.2.3.4
port: 443
password: 0123456789abcdef0123456789abcdef
alpn:
  - nw2
up: udp
down: udp
udp: true
`)

	if vendor.Status() != interfaces.VStatusOperational {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusOperational, vendor.Status())
	}

	proxyInfo := vendor.ProxyInfo()
	if proxyInfo.Name != "nowhere2" {
		t.Fatalf("expected proxy name nowhere2, got %s", proxyInfo.Name)
	}
	if proxyInfo.Type != interfaces.Nowhere2 {
		t.Fatalf("expected proxy type %s, got %s", interfaces.Nowhere2, proxyInfo.Type)
	}
}

func TestClashBuild_Nowhere3RejectsNw2Options(t *testing.T) {
	vendor := buildClashVendor("nowhere3-invalid", `
name: nowhere3-invalid
type: nowhere
server: 1.2.3.4
port: 443
password: 0123456789abcdef0123456789abcdef
up: udp
down: udp
`)

	if vendor.Status() != interfaces.VStatusNotReady {
		t.Fatalf("expected status %v, got %v", interfaces.VStatusNotReady, vendor.Status())
	}
}
