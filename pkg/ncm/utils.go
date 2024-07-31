// MIT License
//
// Copyright (c) 2024 chaunsin
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

package ncm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type CoverType string

const (
	CoverTypeUnknown CoverType = "unknown"
	CoverTypePng     CoverType = "png"
	CoverTypeJpeg    CoverType = "jpeg"
)

func (c CoverType) FileType() string {
	return string(c)
}

func (c CoverType) MIME() string {
	switch c {
	case CoverTypeJpeg:
		return "image/jpeg"
	case CoverTypePng:
		return "image/png"
	case CoverTypeUnknown:
		fallthrough
	default:
		return "unknown"
	}
}

var (
	pngPrefix  = []byte("\x89PNG\x0D\x0A\x1A\x0A")
	jpegPrefix = []byte("\xFF\xD8\xFF")
)

func DetectCoverType(data []byte) CoverType {
	if bytes.HasPrefix(data, jpegPrefix) {
		return CoverTypeJpeg
	}
	if bytes.HasPrefix(data, pngPrefix) {
		return CoverTypePng
	}
	return CoverTypeUnknown
}

func readUint32(rBuf []byte, rs io.ReadSeeker) (uint32, error) {
	if n, err := rs.Read(rBuf); err != nil {
		return uint32(n), fmt.Errorf("read: %w", err)
	}
	return binary.LittleEndian.Uint32(rBuf), nil
}
