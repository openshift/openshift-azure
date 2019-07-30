package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"os"
	"time"

	uuid "github.com/satori/go.uuid"
)

var (
	size = flag.Uint64("size", 0, "image size in bytes")
)

// from
// http://download.microsoft.com/download/f/f/e/ffef50a5-07dd-4cf8-aaa3-442c0673a029/Virtual%20Hard%20Disk%20Format%20Spec_10_18_06.doc

type vhdFooter struct {
	Cookie             [8]byte
	Features           uint32
	FileFormatVersion  uint32
	DataOffset         uint64
	TimeStamp          uint32
	CreatorApplication [4]byte
	CreatorVersion     uint32
	CreatorHostOS      [4]byte
	OriginalSize       uint64
	CurrentSize        uint64
	DiskGeometry       uint32
	DiskType           uint32
	Checksum           uint32
	UniqueID           uuid.UUID
	SavedState         uint8
	Reserved           [427]byte
}

func (f *vhdFooter) CalculateChecksum() error {
	f.Checksum = 0

	buf := &bytes.Buffer{}

	err := binary.Write(buf, binary.BigEndian, f)
	if err != nil {
		return err
	}

	for _, b := range buf.Bytes() {
		f.Checksum += uint32(b)
	}
	f.Checksum = ^f.Checksum

	return nil
}

func main() {
	flag.Parse()

	if *size == 0 {
		flag.Usage()
		os.Exit(2)
	}

	id := uuid.NewV4()

	f := vhdFooter{
		Features:          2,          // reserved
		FileFormatVersion: 0x00010000, // 1.0
		DataOffset:        0xffffffffffffffff,
		TimeStamp:         uint32(time.Now().Sub(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) / time.Second),
		CreatorVersion:    0x00050003, // 5.3
		OriginalSize:      *size,
		CurrentSize:       *size,
		DiskGeometry:      0xffff10ff, // 65535 cylinders, 16 heads, 255 sectors
		DiskType:          2,          // fixed
		UniqueID:          id,
	}

	copy(f.Cookie[:], []byte("conectix"))
	copy(f.CreatorApplication[:], []byte("qem2"))
	copy(f.CreatorHostOS[:], []byte("Wi2k"))

	err := f.CalculateChecksum()
	if err != nil {
		panic(err)
	}

	err = binary.Write(os.Stdout, binary.BigEndian, f)
	if err != nil {
		panic(err)
	}
}
