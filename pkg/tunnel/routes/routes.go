package routes

import (
	"log"
	"net"
	"syscall"
	"unsafe"
)

func Add(nets []net.IPNet, iface string) error {
	for _, n := range nets {
		err := route(syscall.SIOCADDRT, n, iface)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func Delete(nets []net.IPNet, iface string) error {
	for _, n := range nets {
		err := route(syscall.SIOCDELRT, n, iface)
		if err != nil {
			log.Println(err)
			return err
		}
	}

	return nil
}

func route(ioctl uintptr, n net.IPNet, iface string) error {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(sock)

	var rtentry struct {
		rtPad1    uint64
		rtDst     syscall.RawSockaddrInet4
		rtGateway syscall.RawSockaddrInet4
		rtGenmask syscall.RawSockaddrInet4
		rtFlags   uint16
		rtPad2    uint16
		rtPad3    uint64
		rtTos     uint8
		rtClass   uint8
		rtPad4    [3]uint16
		rtMetric  int16
		rtDev     uintptr
		rtMtu     uint64
		rtWindow  uint64
		rtIrtt    uint16
	}

	rtentry.rtDst.Family = syscall.AF_INET
	copy(rtentry.rtDst.Addr[:], n.IP.To4())

	rtentry.rtGenmask.Family = syscall.AF_INET
	copy(rtentry.rtGenmask.Addr[:], net.IP(n.Mask).To4())

	rtdev := [syscall.IFNAMSIZ]byte{}
	copy(rtdev[:], []byte(iface))
	rtdev[len(rtdev)-1] = 0
	rtentry.rtDev = uintptr(unsafe.Pointer(&rtdev))

	rtentry.rtFlags = syscall.RTF_UP

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), ioctl, uintptr(unsafe.Pointer(&rtentry)))
	if errno != 0 {
		return syscall.Errno(errno)
	}

	return nil
}
