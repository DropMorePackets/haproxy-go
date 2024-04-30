// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This code is copied from https://github.com/criteo/haproxy-spoe-go/blob/master/encoding.go

package encoding

import (
	"fmt"
	"io"
)

var (
	ErrUnterminatedSequence = fmt.Errorf("unterminated sequence")
	ErrInsufficientSpace    = fmt.Errorf("insufficient space in buffer")
)

func ReadVarint(rd io.ByteReader) (uint64, error) {
	b, err := rd.ReadByte()
	if err != nil {
		return 0, ErrUnterminatedSequence
	}

	val := uint64(b)
	off := 1

	if val < 240 {
		return val, nil
	}

	r := uint(4)
	for {
		b, err := rd.ReadByte()
		if err != nil {
			return 0, ErrUnterminatedSequence
		}

		v := uint64(b)
		val += v << r
		off++
		r += 7

		if v < 128 {
			break
		}
	}

	return val, nil
}

func PutVarint(b []byte, i uint64) (int, error) {
	if len(b) == 0 {
		return 0, ErrInsufficientSpace
	}

	if i < 240 {
		b[0] = byte(i)
		return 1, nil
	}

	n := 0
	b[n] = byte(i) | 240
	n++
	i = (i - 240) >> 4
	for i >= 128 {
		if n > len(b)-1 {
			return 0, ErrInsufficientSpace
		}

		b[n] = byte(i) | 128
		n++
		i = (i - 128) >> 7
	}

	if n > len(b)-1 {
		return 0, ErrInsufficientSpace
	}

	b[n] = byte(i)
	n++

	return n, nil
}

func Varint(b []byte) (uint64, int, error) {
	if len(b) == 0 {
		return 0, 0, ErrUnterminatedSequence
	}
	val := uint64(b[0])
	off := 1

	if val < 240 {
		return val, 1, nil
	}

	r := uint(4)
	for {
		if off > len(b)-1 {
			return 0, 0, ErrUnterminatedSequence
		}

		v := uint64(b[off])
		val += v << r
		off++
		r += 7

		if v < 128 {
			break
		}
	}

	return val, off, nil
}
