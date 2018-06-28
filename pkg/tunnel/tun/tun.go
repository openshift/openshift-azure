package tun

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

func Tun(iface string) (io.ReadWriteCloser, error) {
	err := os.RemoveAll("tun")
	if err != nil {
		return nil, err
	}

	err = syscall.Mknod("tun", syscall.S_IFCHR|0666, 10<<8|200)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile("tun", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	ifreq := struct {
		ifrName  [syscall.IFNAMSIZ]byte
		ifrFlags int16
	}{}
	copy(ifreq.ifrName[:], []byte(iface))
	ifreq.ifrFlags = syscall.IFF_TUN | syscall.IFF_NO_PI

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), syscall.TUNSETIFF, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return nil, syscall.Errno(errno)
	}

	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(sock)

	ifreq.ifrFlags = syscall.IFF_UP | syscall.IFF_POINTOPOINT | syscall.IFF_RUNNING | syscall.IFF_NOARP | syscall.IFF_MULTICAST

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFFLAGS, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return nil, syscall.Errno(errno)
	}

	return f, nil
}
