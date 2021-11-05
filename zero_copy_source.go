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
	"encoding/binary"
)

type ZeroCopySource struct {
	s   []byte
	off uint64 // current reading index
}

// Len returns the number of bytes of the unread portion of the
// slice.
func (z *ZeroCopySource) Len() uint64 {
	length := uint64(len(z.s))
	if z.off >= length {
		return 0
	}
	return length - z.off
}

func (z *ZeroCopySource) Bytes() []byte {
	return z.s
}

func (z *ZeroCopySource) OffBytes() []byte {
	return z.s[z.off:]
}

func (z *ZeroCopySource) Pos() uint64 {
	return z.off
}

// Size returns the original length of the underlying byte slice.
// Size is the number of bytes available for reading via ReadAt.
// The returned value is always the same and is not affected by calls
// to any other method.
func (z *ZeroCopySource) Size() uint64 { return uint64(len(z.s)) }

// NextBytes Read implements the io.ZeroCopySource interface.
func (z *ZeroCopySource) NextBytes(n uint64) (data []byte, eof bool) {
	m := uint64(len(z.s))
	end, overflow := SafeAdd(z.off, n)
	if overflow || end > m {
		end = m
		eof = true
	}
	data = z.s[z.off:end]
	z.off = end

	return
}

func (z *ZeroCopySource) Skip(n uint64) (eof bool) {
	m := uint64(len(z.s))
	end, overflow := SafeAdd(z.off, n)
	if overflow || end > m {
		end = m
		eof = true
	}
	z.off = end

	return
}

// NextByte ReadByte implements the io.ByteReader interface.
func (z *ZeroCopySource) NextByte() (data byte, eof bool) {
	if z.off >= uint64(len(z.s)) {
		return 0, true
	}

	b := z.s[z.off]
	z.off++
	return b, false
}

func (z *ZeroCopySource) NextUint8() (data uint8, eof bool) {
	var val byte
	val, eof = z.NextByte()
	return uint8(val), eof
}

func (z *ZeroCopySource) NextBool() (data bool, eof bool) {
	val, eof := z.NextByte()
	if val == 0 {
		data = false
	} else if val == 1 {
		data = true
	} else {
		eof = true
	}
	return
}

// BackUp Backs up a number of bytes, so that the next call to NextXXX() returns data again
// that was already returned by the last call to NextXXX().
func (z *ZeroCopySource) BackUp(n uint64) {
	z.off -= n
}

func (z *ZeroCopySource) NextUint16() (data uint16, eof bool) {
	var buf []byte
	buf, eof = z.NextBytes(Uint16Size)
	if eof {
		return
	}

	return binary.LittleEndian.Uint16(buf), eof
}

func (z *ZeroCopySource) NextUint32() (data uint32, eof bool) {
	var buf []byte
	buf, eof = z.NextBytes(Uint32Size)
	if eof {
		return
	}

	return binary.LittleEndian.Uint32(buf), eof
}

func (z *ZeroCopySource) NextUint64() (data uint64, eof bool) {
	var buf []byte
	buf, eof = z.NextBytes(Uint64Size)
	if eof {
		return
	}

	return binary.LittleEndian.Uint64(buf), eof
}

func (z *ZeroCopySource) NextInt32() (data int32, eof bool) {
	var val uint32
	val, eof = z.NextUint32()
	return int32(val), eof
}

func (z *ZeroCopySource) NextInt64() (data int64, eof bool) {
	var val uint64
	val, eof = z.NextUint64()
	return int64(val), eof
}

func (z *ZeroCopySource) NextInt16() (data int16, eof bool) {
	var val uint16
	val, eof = z.NextUint16()
	return int16(val), eof
}

func (z *ZeroCopySource) NextVarBytes() (data []byte, eof bool) {
	count, eof := z.NextVarUint()
	if eof {
		return
	}
	data, eof = z.NextBytes(count)
	return
}

func (z *ZeroCopySource) NextAddress() (data Address, eof bool) {
	var buf []byte
	buf, eof = z.NextBytes(AddrLen)
	if eof {
		return
	}
	copy(data[:], buf)

	return
}

func (z *ZeroCopySource) NextHash() (data Uint256, eof bool) {
	var buf []byte
	buf, eof = z.NextBytes(Uint256Size)
	if eof {
		return
	}
	copy(data[:], buf)

	return
}

func (z *ZeroCopySource) NextString() (data string, eof bool) {
	var val []byte
	val, eof = z.NextVarBytes()
	data = string(val)
	return
}

func (z *ZeroCopySource) NextVarUint() (data uint64, eof bool) {
	var fb byte
	fb, eof = z.NextByte()
	if eof {
		return
	}

	switch fb {
	case 0xFD:
		val, e := z.NextUint16()
		if e {
			eof = e
			return
		}
		data = uint64(val)
	case 0xFE:
		val, e := z.NextUint32()
		if e {
			eof = e
			return
		}
		data = uint64(val)
	case 0xFF:
		val, e := z.NextUint64()
		if e {
			eof = e
			return
		}
		data = uint64(val)
	default:
		data = uint64(fb)
	}
	return
}

// NewZeroCopySource NewReader returns a new ZeroCopySource reading from b.
func NewZeroCopySource(b []byte) *ZeroCopySource { return &ZeroCopySource{b, 0} }
