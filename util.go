package goembed

import (
	"bytes"
	"hash/crc32"
)

var hex = [16]byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'}
var crc32c_table = crc32.MakeTable(crc32.Castagnoli)

func crc32c(data []byte) string {
	crc := crc32.Update(0, crc32c_table, data)
	buf := make([]byte, 8)
	buf[0] = hex[(crc>>28)&0x0f]
	buf[1] = hex[(crc>>24)&0x0f]
	buf[2] = hex[(crc>>20)&0x0f]
	buf[3] = hex[(crc>>16)&0x0f]
	buf[4] = hex[(crc>>12)&0x0f]
	buf[5] = hex[(crc>>8)&0x0f]
	buf[6] = hex[(crc>>4)&0x0f]
	buf[7] = hex[(crc)&0x0f]
	return string(buf)
}

func indent(w *bytes.Buffer, n int) {
	for i := 0; i < n; i++ {
		w.WriteByte('\t')
	}
}
