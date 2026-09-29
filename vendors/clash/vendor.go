package clash

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"

	"github.com/metacubex/mihomo/component/resolver"
	"github.com/metacubex/mihomo/constant"
	"github.com/miaokobot/miaospeed/interfaces"
	"github.com/miaokobot/miaospeed/utils"
)

type Clash struct {
	proxy      constant.Proxy
	proxyName  string
	proxyInfo  string
	dnsServers []string
}

func setupIPv6() {
	if utils.GCFG.EnableIPv6 && resolver.DisableIPv6 {
		resolver.DisableIPv6 = false
	}
}

func (c *Clash) Type() interfaces.VendorType {
	return interfaces.VendorClash
}

func (c *Clash) Status() interfaces.VendorStatus {
	if c == nil || c.proxy == nil {
		return interfaces.VStatusNotReady
	}

	return interfaces.VStatusOperational
}

func (c *Clash) Build(proxyName string, proxyInfo string) interfaces.Vendor {
	if c == nil {
		c = &Clash{}
	}
	c.proxyName = proxyName
	c.proxyInfo = proxyInfo
	c.rebuildProxy()
	return c
}

func (c *Clash) SetDNSServers(servers []string) {
	if c == nil {
		return
	}
	c.dnsServers = append([]string(nil), servers...)
	utils.DLogf("Clash DNS servers updated | proxyName=%q | dnsServers=%d", c.proxyName, len(c.dnsServers))
	c.rebuildProxy()
}

func (c *Clash) rebuildProxy() {
	if c == nil {
		return
	}
	if strings.TrimSpace(c.proxyInfo) == "" {
		c.proxy = nil
		return
	}
	proxy := extractFirstProxyWithDNS(c.proxyName, c.proxyInfo, c.dnsServers)
	if proxy != nil {
		c.proxy = proxy
	}
}

func pickResolvedIPForNetwork(ips []net.IP, network interfaces.RequestOptionsNetwork) (netip.Addr, bool) {
	if network == interfaces.ROptionsTCP6 {
		for _, ip := range ips {
			addr, ok := netip.AddrFromSlice(ip)
			if !ok {
				continue
			}
			addr = addr.Unmap()
			if addr.Is6() {
				return addr, true
			}
		}
		return netip.Addr{}, false
	}

	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			continue
		}
		addr = addr.Unmap()
		if addr.Is4() {
			return addr, true
		}
	}

	for _, ip := range ips {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok {
			continue
		}
		addr = addr.Unmap()
		if addr.Is6() {
			return addr, true
		}
	}

	return netip.Addr{}, false
}

func (c *Clash) applyMihomoDNSForDial(addr *constant.Metadata, network interfaces.RequestOptionsNetwork) {
	if c == nil || addr == nil || addr.Host == "" || len(c.dnsServers) == 0 {
		return
	}
	if _, err := netip.ParseAddr(addr.Host); err == nil {
		return
	}

	ips, matched, err := utils.LookupByMihomoDNSServers(addr.Host, c.dnsServers)
	if !matched || err != nil {
		return
	}
	resolvedIP, ok := pickResolvedIPForNetwork(ips, network)
	if !ok {
		return
	}

	addr.DstIP = resolvedIP
	addr.Host = ""
}

func (c *Clash) DialTCP(ctx context.Context, url string, network interfaces.RequestOptionsNetwork) (net.Conn, error) {
	if c == nil || c.proxy == nil {
		return nil, fmt.Errorf("should call Build() before run")
	}
	setupIPv6()

	addr, err := urlToMetadata(url, constant.TCP)
	if err != nil {
		return nil, fmt.Errorf("cannot build tcp context")
	}
	c.applyMihomoDNSForDial(&addr, network)

	return c.proxy.DialContext(ctx, &addr)
}

func (c *Clash) DialUDP(ctx context.Context, url string) (net.PacketConn, error) {
	if c == nil || c.proxy == nil {
		return nil, fmt.Errorf("should call Build() before run")
	}
	setupIPv6()

	addr, err := urlToMetadata(url, constant.UDP)
	if err != nil {
		return nil, fmt.Errorf("cannot build udp context")
	}

	return c.proxy.ListenPacketContext(ctx, &addr)

}
func (c *Clash) ProxyInfo() interfaces.ProxyInfo {
	if c == nil || c.proxy == nil {
		return interfaces.ProxyInfo{}
	}

	return interfaces.ProxyInfo{
		Name:    c.proxy.Name(),
		Address: c.proxy.Addr(),
		Type:    interfaces.Parse(c.proxy.Type().String()),
	}
}
