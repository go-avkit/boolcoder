// Copyright (c) 2026, go-avkit
// SPDX-License-Identifier: BSD-3-Clause

package boolcoder

import "testing"

// FuzzDecoder drives a Decoder over arbitrary bytes, with the probabilities and
// widths chosen by the input as well, and holds it to what a reader of
// downloaded frames needs: no panic, a range that stays a byte in its upper
// half, results no wider than asked for, and an exhaustion that is reported
// rather than hidden once the data has run out.
//
//	go test -run='^$' -fuzz='^FuzzDecoder$' -fuzztime=60s
func FuzzDecoder(f *testing.F) {
	f.Add([]byte{}, []byte{128})
	f.Add([]byte{0x9D, 0x01, 0x2A, 0x40, 0x00, 0xF0}, []byte{1, 2, 3, 255, 0, 128})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF}, []byte{33, 64, 200, 7, 7, 7, 7})
	f.Fuzz(func(t *testing.T, data, ops []byte) {
		d := NewDecoder(data)
		for _, op := range ops {
			switch op % 3 {
			case 0:
				if b := d.Bool(op); b > 1 {
					t.Fatalf("Bool returned %d", b)
				}
			case 1:
				n := int(op>>2) % 33
				if v := d.Literal(n); n < 32 && v>>uint(n) != 0 {
					t.Fatalf("Literal(%d) returned %#x, wider than asked", n, v)
				}
			case 2:
				_ = d.Signed(int(op>>2) % 32)
			}
			if d.rng < 128 || d.rng > 255 {
				t.Fatalf("range %d left its byte", d.rng)
			}
		}
		if d.pos > len(data) {
			t.Fatalf("position %d past %d bytes", d.pos, len(data))
		}
		// Zeros are shifted in only once the data has run out, so a decoder
		// that says it ran short must have consumed every byte.
		if d.Err() != nil && d.pos != len(data) {
			t.Fatalf("exhausted at byte %d of %d", d.pos, len(data))
		}
	})
}
