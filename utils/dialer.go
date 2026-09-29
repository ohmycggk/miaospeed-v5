package utils

import (
	"context"
	"fmt"
	"net"
	"strings"
)

func ValidateNetworkInterface(name string) error {
	if name == "" {
		return nil
	}

	itf, err := net.InterfaceByName(name)
	if err != nil {
		return fmt.Errorf("cannot find network interface %q: %w", name, err)
	}

	if addrs, err := itf.Addrs(); err != nil {
		return fmt.Errorf("cannot read addresses of network interface %q: %w", name, err)
	} else if len(addrs) == 0 {
		return fmt.Errorf("network interface %q has no address", name)
	}

	return nil
}

func Dial(network string, address string) (net.Conn, error) {
	return DialContext(context.Background(), network, address)
}

func DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	dialer := net.Dialer{}
	if err := bindDialerLocalAddr(&dialer, network); err != nil {
		return nil, err
	}

	return dialer.DialContext(ctx, network, address)
}

func bindDialerLocalAddr(dialer *net.Dialer, network string) error {
	if dialer == nil || GCFG.NetworkInterface == "" {
		return nil
	}

	ip, err := pickInterfaceIP(GCFG.NetworkInterface, network)
	if err != nil {
		return err
	}

	if strings.HasPrefix(network, "udp") {
		dialer.LocalAddr = &net.UDPAddr{IP: ip}
		return nil
	}
	dialer.LocalAddr = &net.TCPAddr{IP: ip}
	return nil
}

func pickInterfaceIP(interfaceName string, network string) (net.IP, error) {
	itf, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return nil, fmt.Errorf("cannot find network interface %q: %w", interfaceName, err)
	}

	addrs, err := itf.Addrs()
	if err != nil {
		return nil, fmt.Errorf("cannot read addresses of network interface %q: %w", interfaceName, err)
	}

	var ipv4Candidate net.IP
	var ipv6Candidate net.IP
	for _, addr := range addrs {
		var ip net.IP

		switch v := addr.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}

		if ip == nil || ip.IsUnspecified() {
			continue
		}

		if v4 := ip.To4(); v4 != nil {
			if strings.HasSuffix(network, "4") {
				return v4, nil
			}
			if ipv4Candidate == nil {
				ipv4Candidate = v4
			}
			continue
		}

		if ip.To16() != nil {
			if strings.HasSuffix(network, "6") {
				return ip, nil
			}
			if ipv6Candidate == nil {
				ipv6Candidate = ip
			}
		}
	}

	if strings.HasSuffix(network, "4") {
		return nil, fmt.Errorf("network interface %q has no ipv4 address", interfaceName)
	}
	if strings.HasSuffix(network, "6") {
		return nil, fmt.Errorf("network interface %q has no ipv6 address", interfaceName)
	}
	if ipv4Candidate != nil {
		return ipv4Candidate, nil
	}
	if ipv6Candidate != nil {
		return ipv6Candidate, nil
	}

	return nil, fmt.Errorf("network interface %q has no usable ip address", interfaceName)
}
