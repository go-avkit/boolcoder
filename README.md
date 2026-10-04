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

## Licence

BSD-3-Clause.
