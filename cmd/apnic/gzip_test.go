package main

import (
	"compress/gzip"
	"strings"
)

// gzipCompress gzip-compresses data, mirroring the SDK test helper.
func gzipCompress(data []byte) []byte {
	var buf strings.Builder
	zw := gzip.NewWriter(&buf)
	zw.Write(data)
	zw.Close()
	return []byte(buf.String())
}
