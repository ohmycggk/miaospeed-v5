package clash

import (
	"net"
	"strings"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/transport/nowhere"
	"github.com/miaokobot/miaospeed/interfaces"
	"github.com/miaokobot/miaospeed/utils"
	vendorlog "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func init() {
	logger := vendorlog.StandardLogger()
	logger.ExitFunc = func(code int) {}
}

func pickResolvedProxyServerIP(ips []net.IP) (string, bool) {
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String(), true
		}
	}
	for _, ip := range ips {
		if ip.To16() != nil {
			return ip.String(), true
		}
	}
	return "", false
}

func setStringIfMissing(payload map[string]any, key string, val string) {
	if payload == nil || strings.TrimSpace(val) == "" {
		return
	}
	if existing, ok := payload[key]; ok {
		if s, sok := existing.(string); sok && strings.TrimSpace(s) != "" {
			return
		}
	}
	payload[key] = val
}

func applyMihomoDNSForProxyServer(payload map[string]any, dnsServers []string) {
	if len(dnsServers) == 0 || payload == nil {
		return
	}

	rawServer, ok := payload["server"]
	if !ok {
		return
	}
	server, ok := rawServer.(string)
	if !ok {
		return
	}
	server = strings.TrimSpace(server)
	if server == "" || net.ParseIP(server) != nil {
		return
	}

	ips, matched, err := utils.LookupByMihomoDNSServers(server, dnsServers)
	if !matched {
		return
	}
	if err != nil {
		utils.DLogf("Mihomo DNS resolve proxy server failed | server=%q | err=%v", server, err)
		return
	}

	resolvedIP, ok := pickResolvedProxyServerIP(ips)
	if !ok {
		utils.DLogf("Mihomo DNS resolved proxy server but no IP selected | server=%q", server)
		return
	}

	payload["server"] = resolvedIP
	setStringIfMissing(payload, "sni", server)
	setStringIfMissing(payload, "servername", server)
}

// nowhereProxyType reports which Nowhere stack a nowhere payload selects, using
// the core's own ALPN rule (nowhere.SelectNowhereStack): an omitted ALPN or a
// single nw3 is Nowhere 3, exactly nw2 is the Nowhere 2 compatibility stack.
// The core rejects every other ALPN at parse time, so a nowhere proxy that
// builds successfully always resolves here. Non-nowhere payloads return "".
func nowhereProxyType(payload map[string]any) interfaces.ProxyType {
	if proxyType, _ := payload["type"].(string); proxyType != "nowhere" {
		return ""
	}

	var alpn []string
	switch v := payload["alpn"].(type) {
	case []any:
		for _, item := range v {
			if name, ok := item.(string); ok {
				alpn = append(alpn, name)
			}
		}
	case []string:
		alpn = v
	case string:
		alpn = []string{v}
	}

	nw3, err := nowhere.SelectNowhereStack(alpn)
	if err != nil {
		return ""
	}
	if nw3 {
		return interfaces.Nowhere3
	}
	return interfaces.Nowhere2
}

func parseProxy(proxyName, proxyPayload string, dnsServers []string) (constant.Proxy, interfaces.ProxyType) {
	var payload map[string]any
	if err := yaml.Unmarshal([]byte(proxyPayload), &payload); err != nil {
		utils.DLogf("Vendor Parser | Parse clash profile yaml error, error=%v", err.Error())
		return nil, ""
	}
	if payload == nil {
		utils.DLogf("Vendor Parser | Parse clash profile error, empty payload")
		return nil, ""
	}
	if _, ok := payload["name"]; !ok {
		payload["name"] = proxyName
	}
	applyMihomoDNSForProxyServer(payload, dnsServers)
	proxy, err := adapter.ParseProxy(payload)

	if err != nil {
		utils.DLogf("Vendor Parser | Parse clash profile error, error=%v", err.Error())
	}

	return proxy, nowhereProxyType(payload)
}

func extractFirstProxy(proxyName, proxyPayload string) (constant.Proxy, interfaces.ProxyType) {
	return extractFirstProxyWithDNS(proxyName, proxyPayload, nil)
}

func extractFirstProxyWithDNS(proxyName, proxyPayload string, dnsServers []string) (constant.Proxy, interfaces.ProxyType) {
	proxy, nowhereType := parseProxy(proxyName, proxyPayload, dnsServers)

	if proxy != nil && interfaces.Parse(proxy.Type().String()) != interfaces.ProxyInvalid {
		return proxy, nowhereType
	}

	return nil, ""
}
