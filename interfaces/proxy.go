package interfaces

import "github.com/miaokobot/miaospeed/utils/structs"

type ProxyType string

const (
	Shadowsocks  ProxyType = "Shadowsocks"
	ShadowsocksR ProxyType = "ShadowsocksR"
	Snell        ProxyType = "Snell"
	Socks5       ProxyType = "Socks5"
	Http         ProxyType = "Http"
	Vmess        ProxyType = "Vmess"
	Trojan       ProxyType = "Trojan"

	Vless    ProxyType = "Vless"
	Hysteria ProxyType = "Hysteria"

	Hysteria2   ProxyType = "Hysteria2"
	WireGuard   ProxyType = "WireGuard"
	Tuic        ProxyType = "Tuic"
	Ssh         ProxyType = "Ssh"
	Mieru       ProxyType = "Mieru"
	AnyTLS      ProxyType = "AnyTLS"
	Sudoku      ProxyType = "Sudoku"
	Masque      ProxyType = "Masque"
	TrustTunnel ProxyType = "TrustTunnel"
	ShadowQuic  ProxyType = "ShadowQuic"
	OpenVPN     ProxyType = "OpenVPN"
	Tailscale   ProxyType = "Tailscale"
	ZeroTier    ProxyType = "ZeroTier"
	EasyTier    ProxyType = "EasyTier"
	GostRelay   ProxyType = "GostRelay"
	Nowhere     ProxyType = "Nowhere"
	Nowhere2    ProxyType = "Nowhere2"
	Nowhere3    ProxyType = "Nowhere3"

	Direct      ProxyType = "Direct"
	Reject      ProxyType = "Reject"
	RejectDrop  ProxyType = "RejectDrop"
	Compatible  ProxyType = "Compatible"
	Pass        ProxyType = "Pass"
	PassRule    ProxyType = "PassRule"
	Rematch     ProxyType = "Rematch"
	Dns         ProxyType = "Dns"
	Relay       ProxyType = "Relay"
	Selector    ProxyType = "Selector"
	Fallback    ProxyType = "Fallback"
	URLTest     ProxyType = "URLTest"
	LoadBalance ProxyType = "LoadBalance"

	ProxyInvalid ProxyType = "Invalid"
)

var AllProxyTypes = []ProxyType{
	Shadowsocks, ShadowsocksR, Snell, Socks5, Http, Vmess, Trojan,
	Vless, Hysteria,
	Hysteria2, WireGuard, Tuic, Ssh, Mieru, AnyTLS, Sudoku, Masque,
	TrustTunnel, ShadowQuic, OpenVPN, Tailscale, ZeroTier, EasyTier,
	GostRelay, Nowhere, Nowhere2, Nowhere3,
	Direct, Reject, RejectDrop, Compatible, Pass, PassRule, Rematch, Dns,
	Relay, Selector, Fallback, URLTest, LoadBalance,
}

func Valid(proxyType ProxyType) bool {
	return structs.Contains(AllProxyTypes, proxyType)
}

func Parse(proxyType string) ProxyType {
	pType := ProxyType(proxyType)
	if Valid(pType) {
		return pType
	}
	return ProxyInvalid
}

type ProxyInfo struct {
	Name    string
	Address string
	Type    ProxyType
}

func (pi *ProxyInfo) Map() map[string]string {
	return map[string]string{
		"Name":    pi.Name,
		"Address": pi.Address,
		"Type":    string(pi.Type),
	}
}
