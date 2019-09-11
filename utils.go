package fdfs

import (
	"bytes"
	"io"
	"net"
)

func readCStrFromByteBuffer(buffer *bytes.Buffer, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := buffer.Read(buf); err != nil {
		return "", err
	}

	index := bytes.IndexByte(buf, 0x00)

	if index == -1 {
		return string(buf), nil
	}

	return string(buf[0:index]), nil
}

func writeFromConnToBuffer(conn net.Conn, buffer []byte, size int64) error {
	var (
		err      error
		recv     int
		needRecv int64
	)
	sizeRecv, sizeAll := int64(0), size

	for {
		needRecv = sizeAll - sizeRecv
		if needRecv <= 0 {
			break
		}
		recv, err = conn.Read(buffer[sizeRecv : sizeRecv+needRecv])
		if err != nil {
			return err
		}
		sizeRecv += int64(recv)
	}
	return nil
}

func writeFromConn(conn net.Conn, writer io.Writer, size int64) error {
	var (
		err      error
		recv     int
		needRecv int64
	)
	sizeRecv, sizeAll := int64(0), size
	buf := make([]byte, 4096)

	for {
		needRecv = sizeAll - sizeRecv
		if needRecv <= 0 {
			break
		}
		if needRecv > 4096 {
			needRecv = 4096
		}
		recv, err = conn.Read(buf[:needRecv])
		if err != nil {
			return err
		}
		_, err = writer.Write(buf[:recv])
		if err != nil {
			return err
		}
		sizeRecv += int64(recv)
	}
	return nil
}
