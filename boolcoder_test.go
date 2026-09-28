// Copyright (c) 2026, go-avkit
// SPDX-License-Identifier: BSD-3-Clause

package boolcoder

import (
	"errors"
	"testing"
)

// TestTheTwoSpecificationsStateTheSameSplit.
//
// ⛔ This is what lets one engine serve both codecs, and it is asserted rather
// than believed: a shared engine resting on an unchecked equivalence would be a
// shared defect. Every range a normalised coder can hold, against every
// probability.
func TestTheTwoSpecificationsStateTheSameSplit(t *testing.T) {
	pairs := 0
	for rng := uint32(128); rng < 256; rng++ {
		for p := uint32(0); p < 256; p++ {
			vp8 := 1 + (((rng - 1) * p) >> 8)
			vp9 := (rng*p + (256 - p)) >> 8
			if vp8 != vp9 {
				t.Fatalf("range %d probability %d: VP8 says %d, VP9 says %d", rng, p, vp8, vp9)
			}
			pairs++
		}
	}
	if pairs != 128*256 {
		t.Fatalf("%d pairs tested, want %d", pairs, 128*256)
	}
}

// TestAKnownAnswerDerivedFromTheAlgorithm walks the arithmetic by hand.
//
// ⛔ Derived from the algorithm, not from an encoder. A round trip against an
// encoder written beside it proves only that the two agree, and an encoder wrong
// in the same way passes it -- which is not a remote risk: two attempts at one
// here were both wrong, and the second was wrong in a way the first was not.
//
// The walk, for data 80 00 00 00 read at even chances:
//
//	init      range 255, window 0x80000000 after the first byte
//	symbol 1  split (255*128+128)>>8 = 128, big = 0x80000000, window >= big -> 1
//	          range 127, window 0, normalise to range 254
//	symbol 2  split (254*128+128)>>8 = 127, big = 0x7F000000, window 0 < big -> 0
//	          and the window stays 0, so every symbol after it is 0 too
func TestAKnownAnswerDerivedFromTheAlgorithm(t *testing.T) {
	d := NewDecoder([]byte{0x80, 0x00, 0x00, 0x00})
	if got := d.Bool(128); got != 1 {
		t.Errorf("first symbol = %d, want 1", got)
	}
	for i := 2; i <= 6; i++ {
		if got := d.Bool(128); got != 0 {
			t.Errorf("symbol %d = %d, want 0", i, got)
		}
	}
}

// TestASplitAtTheExtremesStillLeavesARange covers the two probabilities that
// squeeze the split hardest.
//
// ⛔ The range must never reach zero: the normalisation loop would not terminate
// and the next split would divide nothing. The split is at least one and at most
// range minus one, which is what keeps that from happening, and these are the
// probabilities that test it.
func TestASplitAtTheExtremesStillLeavesARange(t *testing.T) {
	for _, p := range []uint8{0, 1, 128, 254, 255} {
		d := NewDecoder([]byte{0xAA, 0x55, 0xF0, 0x0F, 0x12, 0x34})
		for i := 0; i < 40; i++ {
			d.Bool(p)
			if d.rng < 128 || d.rng > 255 {
				t.Fatalf("probability %d, symbol %d: range left at %d", p, i, d.rng)
			}
		}
	}
}

// TestLiteralIsSymbolsAtEvenChances: a literal is n even chances most significant
// first, so it has to agree with reading them one at a time.
func TestLiteralIsSymbolsAtEvenChances(t *testing.T) {
	data := []byte{0x5C, 0xA3, 0x71, 0x0E, 0x92, 0x4D}
	oneAtATime := NewDecoder(data)
	var want uint32
	for i := 0; i < 12; i++ {
		want = want<<1 | oneAtATime.Bool(128)
	}
	if got := NewDecoder(data).Literal(12); got != want {
		t.Errorf("Literal(12) = %012b, want %012b", got, want)
	}
}

// TestSignedIsALiteralAndASign, both ways round, against the same stream read by
// hand.
func TestSignedIsALiteralAndASign(t *testing.T) {
	// ⛔ Several streams, because one gives one sign: a test that met only positive
	// values would leave the negation unexercised, and a sign read the wrong way
	// round is a motion vector pointing into the other half of the picture.
	sawPositive, sawNegative := false, false
	for _, data := range [][]byte{
		{0x5C, 0xA3, 0x71, 0x0E},
		{0xF3, 0x1D, 0x84, 0x60},
		{0x07, 0xC2, 0x39, 0xAB},
		{0x91, 0x4E, 0x6B, 0x22},
	} {
		by := NewDecoder(data)
		magnitude := by.Literal(5)
		negative := by.Bool(128) == 1
		want := int32(magnitude)
		if negative {
			want = -want
			if want != 0 {
				sawNegative = true
			}
		} else if want != 0 {
			sawPositive = true
		}
		if got := NewDecoder(data).Signed(5); got != want {
			t.Errorf("%x: Signed(5) = %d, want %d", data, got, want)
		}
	}
	// The premise, asserted: without both signs occurring this test says half of
	// what it claims to.
	if !sawPositive || !sawNegative {
		t.Errorf("the streams gave positive=%v negative=%v, so one branch went untested",
			sawPositive, sawNegative)
	}
}

// TestReadingPastTheEndIsReportedAtTheEnd.
//
// ⛔ The arithmetic cannot stop mid-symbol -- it needs bits to compare against --
// so past the end it reads zeros and remembers. Err is the only honest place to
// say so, and a decoder that said nothing would hand back a stream of plausible
// symbols invented from nothing.
func TestReadingPastTheEndIsReportedAtTheEnd(t *testing.T) {
	d := NewDecoder([]byte{0x80, 0x40, 0x20, 0x10})
	if err := d.Err(); err != nil {
		t.Fatalf("four bytes are enough to start: %v", err)
	}
	for i := 0; i < 200; i++ {
		d.Bool(128)
	}
	if err := d.Err(); !errors.Is(err, ErrExhausted) {
		t.Errorf("err = %v, want ErrExhausted", err)
	}

	// A decoder given nothing at all is exhausted from the start: the first fill
	// has nothing to take.
	if err := NewDecoder(nil).Err(); !errors.Is(err, ErrExhausted) {
		t.Errorf("empty: err = %v, want ErrExhausted", err)
	}
	// And one given a single byte is not exhausted until it reads past it.
	one := NewDecoder([]byte{0x80})
	if err := one.Err(); err != nil {
		t.Errorf("one byte: %v", err)
	}
}
