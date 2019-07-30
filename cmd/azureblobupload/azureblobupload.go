package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"

	"github.com/Azure/azure-sdk-for-go/storage"
)

var (
	cfgAccountName   = flag.String("account-name", "", "")
	cfgAccountKey    = flag.String("account-key", "", "")
	cfgContainerName = flag.String("container-name", "", "")
	cfgFile          = flag.String("file", "", "")
	cfgName          = flag.String("name", "", "")
)

var empty = make([]byte, 1048576)

type block struct {
	offset int64
	err    error
	data   []byte
}

func run() error {
	f, err := os.Open(*cfgFile)
	if err != nil {
		return err
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		return err
	}

	c, err := storage.NewClient(*cfgAccountName, *cfgAccountKey,
		storage.DefaultBaseURL, storage.DefaultAPIVersion, true)
	if err != nil {
		return err
	}

	bsc := c.GetBlobService()

	ctr := bsc.GetContainerReference(*cfgContainerName)

	_, err = ctr.CreateIfNotExists(nil)
	if err != nil {
		return err
	}

	b := ctr.GetBlobReference(*cfgName)
	b.Properties.ContentLength = st.Size()

	err = b.PutPageBlob(nil)
	if err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	lease, err := b.AcquireLease(-1, "", nil)
	if err != nil {
		return err
	}
	defer b.ReleaseLease(lease, nil)

	blockch := make(chan block)
	go reader(f, st.Size(), blockch)

	errch := make(chan error)
	var i int
	for i = 0; i < 4; i++ {
		go writer(b, lease, blockch, errch)
	}

out:
	for i > 0 {
		select {
		case err = <-errch:
			if err != nil {
				break
			}
			i--
		case <-sig:
			break out
		}
	}

	return err
}

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func reader(f *os.File, sz int64, blockch chan<- block) {
	for pos := int64(0); pos < sz; {
		data := make([]byte, min(int64(len(empty)), sz-pos))

		_, err := io.ReadFull(f, data)
		if err != nil {
			blockch <- block{err: err}
			break
		}

		blockch <- block{offset: pos, data: data}

		pos += int64(len(data))
		fmt.Printf("\r%d/%dMiB (%d%%)", pos/1048576, sz/1048576, 100*pos/sz)
	}

	fmt.Println()
	close(blockch)
}

func writer(b *storage.Blob, lease string, blockch <-chan block, errch chan<- error) {
	var err error

	for err == nil {
		blk, ok := <-blockch
		if !ok || blk.err != nil {
			err = blk.err
			break
		}

		if !bytes.Equal(blk.data, empty[:len(blk.data)]) {
			err = b.WriteRange(storage.BlobRange{
				Start: uint64(blk.offset),
				End:   uint64(blk.offset) + uint64(len(blk.data)) - 1,
			}, bytes.NewBuffer(blk.data),
				&storage.PutPageOptions{LeaseID: lease},
			)
		}
	}

	errch <- err
}

func main() {
	flag.Parse()

	if err := run(); err != nil {
		panic(err)
	}
}
