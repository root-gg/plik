package utils

import (
	"net"
)

func NtoI(ip net.IP) (ipInt uint32) {
	ip = ip.To4()
	ipInt |= uint32(ip[0]) << 24
	ipInt |= uint32(ip[1]) << 16
	ipInt |= uint32(ip[2]) << 8
	ipInt |= uint32(ip[3])
	return
}

func ItoN(ipInt uint32) net.IP {
	bytes := make([]byte, 4)
	bytes[0] = byte(ipInt >> 24 & 0xFF)
	bytes[1] = byte(ipInt >> 16 & 0xFF)
	bytes[2] = byte(ipInt >> 8 & 0xFF)
	bytes[3] = byte(ipInt & 0xFF)
	return net.IP(bytes)
}
