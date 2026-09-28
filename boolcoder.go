// Copyright (c) 2026, go-avkit
// SPDX-License-Identifier: BSD-3-Clause

// Package boolcoder decodes the boolean arithmetic coding that VP8 and VP9 carry
// everything but their frame headers in, in pure Go.
//
// It is to the VPx family what a bit reader is to the H.26x family: the one
// mechanism every part above it goes through, shared so that neither codec has to
// own it and neither depends on the other.
//
// ⛔ One engine serves both, and that is measured rather than assumed. The two
// specifications state the split point differently --
//
//	VP8: 1 + (((range-1) * probability) >> 8)
//	VP9: (range * probability + (256 - probability)) >> 8
//
// -- and they are the same expression: adding 256 before a shift of eight is
// adding one after it, so the two agree for every one of the 32768 pairs of range
// and probability that can occur. A test asserts that over all of them, because a
// shared engine resting on an unchecked equivalence would be a shared defect.
//
// AV1 is NOT served by this. Its coder carries probabilities on fifteen bits and
// normalises differently; it belongs beside this rather than inside it.
package boolcoder

import "errors"

// valueBits is the width of the window the arithmetic value lives in.
const valueBits = 32

// ErrExhausted means the decoder read past the end of its data.
//
// It is reported rather than hidden because a bitstream that ends early is a
// bitstream that was cut, and a decoder handed zeros past the end would go on
// producing plausible symbols from nothing.
var ErrExhausted = errors.New("boolcoder: read past the end of the partition")

// Decoder reads symbols from one arithmetic-coded partition.
type Decoder struct {
	data  []byte
	pos   int
	value uint32 // the window, with the arithmetic value in its high bits
	rng   uint32 // 128 to 255
	count int    // bits of value below the high byte that are still valid
	short bool   // a read has gone past the end
}

// NewDecoder starts reading data.
//
// The state it begins in is what the format states: a range of the whole byte, an
// empty window, and a count that makes the first read fill it.
func NewDecoder(data []byte) *Decoder {
	d := &Decoder{data: data, rng: 255, count: -8}
	d.fill()
	return d
}

// fill shifts bytes into the window until it holds enough to read from.
//
// ⛔ Past the end it shifts in zeros and remembers that it did. A decoder cannot
// stop mid-symbol -- the arithmetic needs bits to compare against -- so the
// alternative to zeros is a panic. Remembering is what lets Err report afterwards
// that the symbols from that point on came from nothing.
func (d *Decoder) fill() {
	for d.count < 0 {
		var b uint32
		if d.pos < len(d.data) {
			b = uint32(d.data[d.pos])
			d.pos++
		} else {
			d.short = true
		}
		// ⛔ The window is aligned at the TOP of the word, so the byte goes in
		// just below what is already there and the shift is 16 - count. count is
		// in [-8, -1] here, which makes the shift 17 to 24 -- always positive.
		// Written the other way round, as an offset from the bottom, the shift
		// goes negative as soon as a single bit has been consumed, and Go turns a
		// negative shift into a huge one: the byte is dropped in silence and the
		// decoder goes on producing plausible symbols from nothing.
		d.value |= b << uint(valueBits-8-(d.count+8))
		d.count += 8
	}
}

// Bool reads one bit whose chance of being zero is probability out of 256.
//
// The probability is what the model above this supplies; a probability of 128 is
// an even chance and makes this read a raw bit.
func (d *Decoder) Bool(probability uint8) uint32 {
	// The split is where the range divides between the two outcomes. Written the
	// way VP9 states it; the doc comment shows why VP8's form is the same
	// expression.
	split := (d.rng*uint32(probability) + (256 - uint32(probability))) >> 8
	// The comparison happens at a FIXED place, the top byte of the window, which
	// is what aligning the window at the top buys: no shift here depends on how
	// much has been consumed.
	big := split << (valueBits - 8)

	var bit uint32
	if d.value >= big {
		bit = 1
		d.rng -= split
		d.value -= big
	} else {
		d.rng = split
	}

	// Normalise: bring the range back into the top half of the byte, taking the
	// window with it. The range is never zero -- the split is at least one and at
	// most range-1 -- so this terminates.
	for d.rng < 128 {
		d.rng <<= 1
		d.value <<= 1
		d.count--
	}
	d.fill()
	return bit
}

// Literal reads n raw bits, most significant first: n even chances in a row.
func (d *Decoder) Literal(n int) uint32 {
	var v uint32
	for i := 0; i < n; i++ {
		v = v<<1 | d.Bool(128)
	}
	return v
}

// Signed reads n raw bits and then a sign, which is how these formats state a
// value that can be negative.
func (d *Decoder) Signed(n int) int32 {
	v := int32(d.Literal(n))
	if d.Bool(128) == 1 {
		return -v
	}
	return v
}

// Err reports whether any symbol read so far came from past the end of the data.
//
// It is asked at the END of a partition rather than after each symbol: the
// arithmetic cannot stop mid-symbol, so the question is whether the partition was
// long enough for everything that was read from it, and that is only answerable
// once the reading is done.
func (d *Decoder) Err() error {
	if d.short {
		return ErrExhausted
	}
	return nil
}
