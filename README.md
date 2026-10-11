# boolcoder

Pure-Go (CGO=0) boolean arithmetic decoder — the entropy coder VP8 and VP9 both
read their compressed partitions with.

```go
d := boolcoder.NewDecoder(partition)
flag := d.Bool(128)        // probability out of 256
v := d.Literal(4)          // four bits, each at probability 128
s := d.Signed(6)           // magnitude then sign
if err := d.Err(); err != nil {
    // the partition ended inside a value
}
```

## What is in it

| | |
|---|---|
| `NewDecoder` | a decoder over one compressed partition |
| `Bool(probability)` | one boolean at the given probability out of 256 |
| `Literal(n)` | `n` bits, each at probability 128, most significant first |
| `Signed(n)` | an `n`-bit magnitude followed by a sign bit |
| `Err` | `ErrExhausted` once a read has gone past the end |

**The error is checked at the end, not at each read.** `Bool` returns a bit, not
a bit and an error, because the format reads thousands of them per block and a
decoder that branched on an error after each one would be unreadable. Once the
partition is exhausted the reads return zeroes and `Err` reports it, so the
caller finds out at the boundary it cares about instead of at every bit.

⛔ **A probability here is out of 256**, not a fraction and not out of 255:
`Bool(128)` is an even chance, `Bool(255)` is nearly certain to be false. The
probability is the chance of **zero**, which is the way round the formats state
it and the opposite of what reads naturally.

## Consumers

[`go-avkit/vp8`](https://github.com/go-avkit/vp8) and
[`go-avkit/vp9`](https://github.com/go-avkit/vp9) read the plain bit fields of
their headers; everything behind those is arithmetic-coded, and this is what
reads it.

**Windows, macOS, Linux; six 64-bit architectures.** 100% statement coverage,
gated in CI.

## A width the result cannot hold

`Literal` returns a `uint32`, so `n` is 0 to 32 and it **panics** outside that.

⛔ It used to shift the first bits out of the `uint32` in silence and return a
value that was not the field asked for — the same shape as
[`bitstream`'s `Bits`](https://github.com/go-avkit/bitstream/pull/7), fixed the
same evening.

A panic is right here and an error is not: **every width in VP8 and VP9 is a
constant of the syntax**, so one outside the range is a mistake in the caller's
code, not something a stream can state. That was measured rather than assumed —
across `go-avkit`, every call to `Literal` and `Signed` passes a constant, and
the only variable one is `Signed`'s own call to `Literal`.

## Licence

BSD-3-Clause.
