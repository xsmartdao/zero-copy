/*
 * Copyright (C) 2018 The ontology Authors
 * This file is part of The ontology library.
 *
 * The ontology is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The ontology is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Lesser General Public License for more details.
 *
 * You should have received a copy of the GNU Lesser General Public License
 * along with The ontology.  If not, see <http://www.gnu.org/licenses/>.
 */

package zero_copy

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type ZeroCopySink struct {
	buf []byte
}

// tryGrowByReSlice is an inlineable version of grow for the fast-case where the
// internal buffer only needs to be re-sliced.
// It returns the index where bytes should be written and whether it succeeded.
func (z *ZeroCopySink) tryGrowByReSlice(n int) (int, bool) {
	if l := len(z.buf); n <= cap(z.buf)-l {
		z.buf = z.buf[:l+n]
		return l, true
	}
	return 0, false
}

const maxInt = int(^uint(0) >> 1)

// grow the buffer to guarantee space for n more bytes.
// It returns the index where bytes should be written.
// If the buffer can't grow it will panic with ErrTooLarge.
func (z *ZeroCopySink) grow(n int) int {
	// Try to grow by means of a reslice.
	if i, ok := z.tryGrowByReSlice(n); ok {
		return i
	}

	l := len(z.buf)
	c := cap(z.buf)
	if c > maxInt-c-n {
		panic(ErrTooLarge)
	}
	// Not enough space anywhere, we need to allocate.
	buf := makeSlice(2*c + n)
	copy(buf, z.buf)
	z.buf = buf[:l+n]
	return l
}

func (z *ZeroCopySink) WriteBytes(p []byte) {
	data := z.NextBytes(uint64(len(p)))
	copy(data, p)
}

func (z *ZeroCopySink) Size() uint64 { return uint64(len(z.buf)) }

func (z *ZeroCopySink) NextBytes(n uint64) (data []byte) {
	m, ok := z.tryGrowByReSlice(int(n))
	if !ok {
		m = z.grow(int(n))
	}
	data = z.buf[m:]
	return
}

// BackUp Backs up a number of bytes, so that the next call to NextXXX() returns data again
// that was already returned by the last call to NextXXX().
func (z *ZeroCopySink) BackUp(n uint64) {
	l := len(z.buf) - int(n)
	z.buf = z.buf[:l]
}

func (z *ZeroCopySink) WriteUint8(data uint8) {
	buf := z.NextBytes(1)
	buf[0] = data
}

func (z *ZeroCopySink) WriteByte(c byte) {
	z.WriteUint8(c)
}

func (z *ZeroCopySink) WriteBool(data bool) {
	if data {
		z.WriteByte(1)
	} else {
		z.WriteByte(0)
	}
}

func (z *ZeroCopySink) WriteUint16(data uint16) {
	buf := z.NextBytes(2)
	binary.LittleEndian.PutUint16(buf, data)
}

func (z *ZeroCopySink) WriteUint32(data uint32) {
	buf := z.NextBytes(4)
	binary.LittleEndian.PutUint32(buf, data)
}

func (z *ZeroCopySink) WriteUint64(data uint64) {
	buf := z.NextBytes(8)
	binary.LittleEndian.PutUint64(buf, data)
}

func (z *ZeroCopySink) WriteInt64(data int64) {
	z.WriteUint64(uint64(data))
}

func (z *ZeroCopySink) WriteInt32(data int32) {
	z.WriteUint32(uint32(data))
}

func (z *ZeroCopySink) WriteInt16(data int16) {
	z.WriteUint16(uint16(data))
}

func (z *ZeroCopySink) WriteVarBytes(data []byte) (size uint64) {
	l := uint64(len(data))
	size = z.WriteVarUint(l) + l

	z.WriteBytes(data)
	return
}

func (z *ZeroCopySink) WriteString(data string) (size uint64) {
	return z.WriteVarBytes([]byte(data))
}

func (z *ZeroCopySink) WriteAddress(addr Address) {
	z.WriteBytes(addr[:])
}

func (z *ZeroCopySink) WriteHash(hash Uint256) {
	z.WriteBytes(hash[:])
}

func (z *ZeroCopySink) WriteVarUint(data uint64) (size uint64) {
	buf := z.NextBytes(9)
	if data < 0xFD {
		buf[0] = uint8(data)
		size = 1
	} else if data <= 0xFFFF {
		buf[0] = 0xFD
		binary.LittleEndian.PutUint16(buf[1:], uint16(data))
		size = 3
	} else if data <= 0xFFFFFFFF {
		buf[0] = 0xFE
		binary.LittleEndian.PutUint32(buf[1:], uint32(data))
		size = 5
	} else {
		buf[0] = 0xFF
		binary.LittleEndian.PutUint64(buf[1:], uint64(data))
		size = 9
	}

	z.BackUp(9 - size)
	return
}

// NewZeroCopySink NewReader returns a new ZeroCopySink reading from b.
func NewZeroCopySink(b []byte) *ZeroCopySink {
	if b == nil {
		b = make([]byte, 0, 512)
	}
	return &ZeroCopySink{b}
}

func (z *ZeroCopySink) Bytes() []byte { return z.buf }

func (z *ZeroCopySink) Reset() { z.buf = z.buf[:0] }

var ErrTooLarge = errors.New("bytes.Buffer: too large")

// makeSlice allocates a slice of size n. If the allocation fails, it panics
// with ErrTooLarge.
func makeSlice(n int) []byte {
	// If the make fails, give a known error.
	defer func() {
		if recover() != nil {
			panic(bytes.ErrTooLarge)
		}
	}()
	return make([]byte, n)
}
