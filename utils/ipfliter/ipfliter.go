package ipfliter

import (
	"net"
	"sync"
)

// IPFilter allows or blocks client addresses. Allowed entries win over blocked
// entries. An empty filter allows every address.
type IPFilter struct {
	mut            sync.RWMutex
	defaultAllowed bool
	ips            map[string]bool
	subnets        []*subnet
}

type subnet struct {
	str     string
	ipnet   *net.IPNet
	allowed bool
}

type Options struct {
	AllowedIPs     []string
	BlockedIPs     []string
	BlockByDefault bool
}

func New(opts Options) *IPFilter {
	f := &IPFilter{
		ips:            map[string]bool{},
		defaultAllowed: !opts.BlockByDefault,
	}
	for _, ip := range opts.BlockedIPs {
		f.ToggleIP(ip, false)
	}
	for _, ip := range opts.AllowedIPs {
		f.ToggleIP(ip, true)
	}
	return f
}

func (f *IPFilter) ToggleIP(str string, allowed bool) bool {
	if ip, network, err := net.ParseCIDR(str); err == nil {
		ones, bits := network.Mask.Size()
		if ones == bits {
			f.mut.Lock()
			f.ips[ip.String()] = allowed
			f.mut.Unlock()
			return true
		}
		f.mut.Lock()
		found := false
		for _, item := range f.subnets {
			if item.str == str {
				item.allowed = allowed
				found = true
				break
			}
		}
		if !found {
			f.subnets = append(f.subnets, &subnet{str: str, ipnet: network, allowed: allowed})
		}
		f.mut.Unlock()
		return true
	}
	if ip := net.ParseIP(str); ip != nil {
		f.mut.Lock()
		f.ips[ip.String()] = allowed
		f.mut.Unlock()
		return true
	}
	return false
}

func (f *IPFilter) Allowed(ipstr string) bool {
	if f == nil {
		return true
	}
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}
	f.mut.RLock()
	defer f.mut.RUnlock()
	if allowed, ok := f.ips[ip.String()]; ok {
		return allowed
	}
	blocked := false
	for _, item := range f.subnets {
		if item.ipnet.Contains(ip) {
			if item.allowed {
				return true
			}
			blocked = true
		}
	}
	if blocked {
		return false
	}
	return f.defaultAllowed
}
