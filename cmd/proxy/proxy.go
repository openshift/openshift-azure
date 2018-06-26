package main

import (
	"io"
	"log"
	"net"
	"os"
	"syscall"
	"unsafe"
)

func setInterfaceIP(iface string, ip, mask net.IP) error {
	sock, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer syscall.Close(sock)

	ifreq := struct {
		ifrName [syscall.IFNAMSIZ]byte
		ifrAddr syscall.RawSockaddrInet4
	}{}
	copy(ifreq.ifrName[:], []byte(iface))
	ifreq.ifrAddr.Family = syscall.AF_INET
	copy(ifreq.ifrAddr.Addr[:], ip.To4())

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFADDR, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return syscall.Errno(errno)
	}

	copy(ifreq.ifrAddr.Addr[:], mask.To4())

	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(sock), syscall.SIOCSIFNETMASK, uintptr(unsafe.Pointer(&ifreq)))
	if errno != 0 {
		return syscall.Errno(errno)
	}

	return nil
}

func proxyOne(done chan<- struct{}, c1, c2 net.Conn) {
	_, err := io.Copy(c1, c2)
	if err != nil {
		log.Println(err)
	}
	if done != nil {
		done <- struct{}{}
	}
}

func proxy(c1 net.Conn) {
	defer c1.Close()

	c2, err := net.Dial("tcp", os.Args[2])
	if err != nil {
		log.Println(err)
		return
	}
	defer c2.Close()

	done := make(chan struct{}, 1)
	go proxyOne(done, c1, c2)
	proxyOne(nil, c2, c1)
	<-done
}

func run() error {
	srcip, _, err := net.SplitHostPort(os.Args[1])
	if err != nil {
		return err
	}

	err = setInterfaceIP("lo:1", net.ParseIP(srcip), net.ParseIP("255.255.255.255"))
	if err != nil {
		return err
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		go proxy(c)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
