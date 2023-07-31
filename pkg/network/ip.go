package network

import (
	"errors"
	"fmt"
	"net"
)

// IsIpV6 returns true if the given string is an IPv6 address
func IsIpV6(addr string) bool {
	ip := net.ParseIP(addr)

	if ip == nil {
		return false
	} else if ip.To4() == nil {
		return true
	} else {
		return false
	}
}

// GetIPv4 returns the IPv4 address of the given ip addr string
func GetIPv4(ipAddr string) (net.IP, error) {
	ip := net.ParseIP(ipAddr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ipAddr)
	} else {
		ipv4 := ip.To4()
		if ipv4 == nil {
			return nil, fmt.Errorf("invalid IP address: %s", ipAddr)
		}
	}

	return ip, nil
}

// GetLocalIpAddr returns the local IP address of the host
// If more than one IP address is found, an error is returned
func GetLocalIpAddr() (net.IP, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil, err
	}

	ipAddresses := make([]net.IP, 0)

	for _, addr := range addrs {
		ipAddr, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		if !ipAddr.IP.IsLoopback() && ipAddr.IP.To4() != nil {
			ipAddresses = append(ipAddresses, ipAddr.IP)
		}
	}

	if len(ipAddresses) > 1 {
		return nil, errors.New("the system has multiple interfaces and could not choose an IP address to advertise. Please specify which one to use with ADVERTISE_ADDR")
	}

	return ipAddresses[0], nil
}
