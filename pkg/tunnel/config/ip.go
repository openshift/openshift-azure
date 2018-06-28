package config

import (
	"net"
	"syscall"
	"unsafe"
)

func getInterfaceIP(iface string) (net.IP, error) {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(sock)

	ifreq := struct {
		ifrName [syscall.IFNAMSIZ]byte
		ifrAddr syscall.RawSockaddrInet4
	}{}
	copy(ifreq.ifrName[:], []byte(iface))

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCGIFADDR, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return nil, syscall.Errno(errno)
	}

	return net.IP(ifreq.ifrAddr.Addr[:]), nil
}
