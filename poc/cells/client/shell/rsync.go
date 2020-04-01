package main

import (
	"fmt"
	"io"

	"bitbucket.org/kardianos/rsync"
	"bitbucket.org/kardianos/rsync/proto"
	"github.com/spf13/afero"
)

func getRsync() *rsync.RSync {
	return &rsync.RSync{
		MaxDataOp: 1024 * 16,
	}
}

func signature(basisFile, sigFile afero.File, blockSizeKiB int) error {
	rs := getRsync()
	rs.BlockSize = 1024 * blockSizeKiB

	defer basisFile.Close()

	defer sigFile.Close()

	sigEncode := &proto.Writer{Writer: sigFile}

	err := sigEncode.Header(proto.TypeSignature, proto.CompNone, rs.BlockSize)
	if err != nil {
		return err
	}
	defer sigEncode.Close()

	return rs.CreateSignature(basisFile, sigEncode.SignatureWriter())
}

func test(basis1File, basis2File afero.File) error {
	defer basis1File.Close()
	defer basis2File.Close()

	basis1Stat, err := basis1File.Stat()
	if err != nil {
		return err
	}
	basis2Stat, err := basis2File.Stat()
	if err != nil {
		return err
	}

	if basis1Stat.Size() != basis2Stat.Size() {
		return fmt.Errorf("File size different.")
	}

	type resetBuffer struct {
		orig, buf []byte
	}

	bufferFount := make(chan resetBuffer, 30)

	b1Source := make(chan resetBuffer, 10)
	b2Source := make(chan resetBuffer, 10)
	errorSource := make(chan error, 4)

	for i := 0; i < cap(bufferFount); i++ {
		b := make([]byte, 32*1024)

		bufferFount <- resetBuffer{
			orig: b,
			buf:  b,
		}
	}

	reader := func(f io.Reader, source chan resetBuffer, errorSource chan error) {
		for {
			buffer := <-bufferFount
			buffer.buf = buffer.orig
			n, err := f.Read(buffer.orig)
			if n == 0 {
				bufferFount <- buffer
			} else {
				buffer.buf = buffer.orig[:n]
				source <- buffer
			}
			if err != nil {
				if err == io.EOF {
					close(source)
					return
				}
				errorSource <- fmt.Errorf("Error reading file: %s", err)
				return
			}
		}
	}

	go reader(basis1File, b1Source, errorSource)
	go reader(basis2File, b2Source, errorSource)

	location := 0
	var b1Buffer resetBuffer
	var b2Buffer resetBuffer
	var ok bool
	for {
		if len(errorSource) > 0 {
			return <-errorSource
		}
		if len(b1Buffer.buf) == 0 {
			if b1Buffer.buf != nil {
				bufferFount <- b1Buffer
			}
			b1Buffer, ok = <-b1Source
			if !ok {
				return nil
			}
		}
		if len(b2Buffer.buf) == 0 {
			if b2Buffer.buf != nil {
				bufferFount <- b2Buffer
			}
			b2Buffer, ok = <-b2Source
			if !ok {
				return nil
			}
		}
		size := min(len(b1Buffer.buf), len(b2Buffer.buf))

		for i := 0; i < size; i++ {
			if b1Buffer.buf[i] != b2Buffer.buf[i] {
				return fmt.Errorf("FAIL: Bytes differ at %d.", location)
			}
			location++
		}
		b1Buffer.buf = b1Buffer.buf[size:]
		b2Buffer.buf = b2Buffer.buf[size:]
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
