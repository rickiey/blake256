// Copyright (c) 2019 The Decred developers
// Originally written in 2011-2012 by Dmitry Chestnykh.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// Package blake256 implements BLAKE-256 and BLAKE-224 hash functions (SHA-3
// candidate).
package blake256

import "hash"

// BlockSize is the block size of the hash algorithm in bytes.
const BlockSize = 64

// Size is the size of BLAKE-256 hash in bytes.
const Size = 32

// Size224 is the size of BLAKE-224 hash in bytes.
const Size224 = 28

type digest struct {
	hashSize int             // hash output size in bits (224 or 256)
	h        [8]uint32       // current chain value
	s        [4]uint32       // salt (zero by default)
	t        uint64          // message bits counter
	nullt    bool            // special case for finalization: skip counter
	x        [BlockSize]byte // buffer for data not yet compressed
	nx       int             // number of bytes in buffer
}

var (
	// Initialization values.
	iv256 = [8]uint32{
		0x6A09E667, 0xBB67AE85, 0x3C6EF372, 0xA54FF53A,
		0x510E527F, 0x9B05688C, 0x1F83D9AB, 0x5BE0CD19}

	iv224 = [8]uint32{
		0xC1059ED8, 0x367CD507, 0x3070DD17, 0xF70E5939,
		0xFFC00B31, 0x68581511, 0x64F98FA7, 0xBEFA4FA4}

	pad = [64]byte{0x80}
)

// Reset resets the state of digest. It leaves salt intact.
func (d *digest) Reset() {
	if d.hashSize == 224 {
		d.h = iv224
	} else {
		d.h = iv256
	}
	d.t = 0
	d.nx = 0
	d.nullt = false
}

func (d *digest) Size() int { return d.hashSize >> 3 }

func (d *digest) BlockSize() int { return BlockSize }

func (d *digest) Write(p []byte) (nn int, err error) {
	nn = len(p)
	if d.nx > 0 {
		n := len(p)
		if n > BlockSize-d.nx {
			n = BlockSize - d.nx
		}
		d.nx += copy(d.x[d.nx:], p)
		if d.nx == BlockSize {
			block(d, d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= BlockSize {
		n := len(p) &^ (BlockSize - 1)
		block(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return
}

// Sum returns the calculated checksum.
func (d digest) Sum(in []byte) []byte {
	// Note d is a copy so that the caller can keep writing and summing.

	sum := d.checkSum()
	if d.Size() == Size224 {
		return append(in, sum[:Size224]...)
	}
	return append(in, sum[:]...)
}

func (d *digest) checkSum() [Size]byte {
	nx := uint64(d.nx)
	l := d.t + nx<<3
	var len [8]byte
	len[0] = byte(l >> 56)
	len[1] = byte(l >> 48)
	len[2] = byte(l >> 40)
	len[3] = byte(l >> 32)
	len[4] = byte(l >> 24)
	len[5] = byte(l >> 16)
	len[6] = byte(l >> 8)
	len[7] = byte(l)

	if nx == 55 {
		// One padding byte.
		d.t -= 8
		if d.hashSize == 224 {
			d.Write([]byte{0x80})
		} else {
			d.Write([]byte{0x81})
		}
	} else {
		if nx < 55 {
			// Enough space to fill the block.
			if nx == 0 {
				d.nullt = true
			}
			d.t -= 440 - nx<<3
			d.Write(pad[0 : 55-nx])
		} else {
			// Need 2 compressions.
			d.t -= 512 - nx<<3
			d.Write(pad[0 : 64-nx])
			d.t -= 440
			d.Write(pad[1:56])
			d.nullt = true
		}
		if d.hashSize == 224 {
			d.Write([]byte{0x00})
		} else {
			d.Write([]byte{0x01})
		}
		d.t -= 8
	}
	d.t -= 64
	d.Write(len[:])

	var out [Size]byte
	j := 0
	for _, s := range d.h[:d.hashSize>>5] {
		out[j+0] = byte(s >> 24)
		out[j+1] = byte(s >> 16)
		out[j+2] = byte(s >> 8)
		out[j+3] = byte(s >> 0)
		j += 4
	}
	return out
}

func (d *digest) setSalt(s []byte) {
	if len(s) != 16 {
		panic("salt length must be 16 bytes")
	}
	d.s[0] = uint32(s[0])<<24 | uint32(s[1])<<16 | uint32(s[2])<<8 | uint32(s[3])
	d.s[1] = uint32(s[4])<<24 | uint32(s[5])<<16 | uint32(s[6])<<8 | uint32(s[7])
	d.s[2] = uint32(s[8])<<24 | uint32(s[9])<<16 | uint32(s[10])<<8 | uint32(s[11])
	d.s[3] = uint32(s[12])<<24 | uint32(s[13])<<16 | uint32(s[14])<<8 | uint32(s[15])
}

// New returns a new hash.Hash computing the BLAKE-256 checksum.
func New() hash.Hash {
	return &digest{
		hashSize: 256,
		h:        iv256,
	}
}

// NewSalt is like New but initializes salt with the given 16-byte slice.
func NewSalt(salt []byte) hash.Hash {
	d := &digest{
		hashSize: 256,
		h:        iv256,
	}
	d.setSalt(salt)
	return d
}

// New224 returns a new hash.Hash computing the BLAKE-224 checksum.
func New224() hash.Hash {
	return &digest{
		hashSize: 224,
		h:        iv224,
	}
}

// New224Salt is like New224 but initializes salt with the given 16-byte slice.
func New224Salt(salt []byte) hash.Hash {
	d := &digest{
		hashSize: 224,
		h:        iv224,
	}
	d.setSalt(salt)
	return d
}

// Sum256 returns the BLAKE-256 checksum of the data.
func Sum256(data []byte) [Size]byte {
	var d digest
	d.hashSize = 256
	d.Reset()
	d.Write(data)
	return d.checkSum()
}

// Sum224 returns the BLAKE-224 checksum of the data.
func Sum224(data []byte) (sum224 [Size224]byte) {
	var d digest
	d.hashSize = 224
	d.Reset()
	d.Write(data)
	sum := d.checkSum()
	copy(sum224[:], sum[:Size224])
	return
}

const (
	cst0  = 0x243F6A88
	cst1  = 0x85A308D3
	cst2  = 0x13198A2E
	cst3  = 0x03707344
	cst4  = 0xA4093822
	cst5  = 0x299F31D0
	cst6  = 0x082EFA98
	cst7  = 0xEC4E6C89
	cst8  = 0x452821E6
	cst9  = 0x38D01377
	cst10 = 0xBE5466CF
	cst11 = 0x34E90C6C
	cst12 = 0xC0AC29B7
	cst13 = 0xC97C50DD
	cst14 = 0x3F84D5B5
	cst15 = 0xB5470917
)

func block(d *digest, p []uint8) {
	h0, h1, h2, h3, h4, h5, h6, h7 := d.h[0], d.h[1], d.h[2], d.h[3], d.h[4], d.h[5], d.h[6], d.h[7]
	s0, s1, s2, s3 := d.s[0], d.s[1], d.s[2], d.s[3]

	for len(p) >= BlockSize {
		v0, v1, v2, v3, v4, v5, v6, v7 := h0, h1, h2, h3, h4, h5, h6, h7
		v8 := cst0 ^ s0
		v9 := cst1 ^ s1
		v10 := cst2 ^ s2
		v11 := cst3 ^ s3
		v12 := uint32(cst4)
		v13 := uint32(cst5)
		v14 := uint32(cst6)
		v15 := uint32(cst7)
		d.t += 512
		if !d.nullt {
			v12 ^= uint32(d.t)
			v13 ^= uint32(d.t)
			v14 ^= uint32(d.t >> 32)
			v15 ^= uint32(d.t >> 32)
		}
		var m [16]uint32

		m[0] = uint32(p[0])<<24 | uint32(p[1])<<16 | uint32(p[2])<<8 | uint32(p[3])
		m[1] = uint32(p[4])<<24 | uint32(p[5])<<16 | uint32(p[6])<<8 | uint32(p[7])
		m[2] = uint32(p[8])<<24 | uint32(p[9])<<16 | uint32(p[10])<<8 | uint32(p[11])
		m[3] = uint32(p[12])<<24 | uint32(p[13])<<16 | uint32(p[14])<<8 | uint32(p[15])
		m[4] = uint32(p[16])<<24 | uint32(p[17])<<16 | uint32(p[18])<<8 | uint32(p[19])
		m[5] = uint32(p[20])<<24 | uint32(p[21])<<16 | uint32(p[22])<<8 | uint32(p[23])
		m[6] = uint32(p[24])<<24 | uint32(p[25])<<16 | uint32(p[26])<<8 | uint32(p[27])
		m[7] = uint32(p[28])<<24 | uint32(p[29])<<16 | uint32(p[30])<<8 | uint32(p[31])
		m[8] = uint32(p[32])<<24 | uint32(p[33])<<16 | uint32(p[34])<<8 | uint32(p[35])
		m[9] = uint32(p[36])<<24 | uint32(p[37])<<16 | uint32(p[38])<<8 | uint32(p[39])
		m[10] = uint32(p[40])<<24 | uint32(p[41])<<16 | uint32(p[42])<<8 | uint32(p[43])
		m[11] = uint32(p[44])<<24 | uint32(p[45])<<16 | uint32(p[46])<<8 | uint32(p[47])
		m[12] = uint32(p[48])<<24 | uint32(p[49])<<16 | uint32(p[50])<<8 | uint32(p[51])
		m[13] = uint32(p[52])<<24 | uint32(p[53])<<16 | uint32(p[54])<<8 | uint32(p[55])
		m[14] = uint32(p[56])<<24 | uint32(p[57])<<16 | uint32(p[58])<<8 | uint32(p[59])
		m[15] = uint32(p[60])<<24 | uint32(p[61])<<16 | uint32(p[62])<<8 | uint32(p[63])

		// Round 1.
		v0 += m[0] ^ cst1
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[2] ^ cst3
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[4] ^ cst5
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[6] ^ cst7
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[5] ^ cst4
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[7] ^ cst6
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[3] ^ cst2
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[1] ^ cst0
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[8] ^ cst9
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[10] ^ cst11
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[12] ^ cst13
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[14] ^ cst15
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[13] ^ cst12
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[15] ^ cst14
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[11] ^ cst10
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[9] ^ cst8
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 2.
		v0 += m[14] ^ cst10
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[4] ^ cst8
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[9] ^ cst15
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[13] ^ cst6
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[15] ^ cst9
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[6] ^ cst13
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[8] ^ cst4
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[10] ^ cst14
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[1] ^ cst12
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[0] ^ cst2
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[11] ^ cst7
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[5] ^ cst3
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[7] ^ cst11
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[3] ^ cst5
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[2] ^ cst0
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[12] ^ cst1
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 3.
		v0 += m[11] ^ cst8
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[12] ^ cst0
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[5] ^ cst2
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[15] ^ cst13
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[2] ^ cst5
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[13] ^ cst15
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[0] ^ cst12
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[8] ^ cst11
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[10] ^ cst14
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[3] ^ cst6
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[7] ^ cst1
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[9] ^ cst4
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[1] ^ cst7
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[4] ^ cst9
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[6] ^ cst3
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[14] ^ cst10
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 4.
		v0 += m[7] ^ cst9
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[3] ^ cst1
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[13] ^ cst12
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[11] ^ cst14
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[12] ^ cst13
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[14] ^ cst11
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[1] ^ cst3
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[9] ^ cst7
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[2] ^ cst6
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[5] ^ cst10
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[4] ^ cst0
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[15] ^ cst8
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[0] ^ cst4
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[8] ^ cst15
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[10] ^ cst5
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[6] ^ cst2
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 5.
		v0 += m[9] ^ cst0
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[5] ^ cst7
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[2] ^ cst4
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[10] ^ cst15
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[4] ^ cst2
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[15] ^ cst10
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[7] ^ cst5
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[0] ^ cst9
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[14] ^ cst1
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[11] ^ cst12
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[6] ^ cst8
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[3] ^ cst13
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[8] ^ cst6
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[13] ^ cst3
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[12] ^ cst11
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[1] ^ cst14
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 6.
		v0 += m[2] ^ cst12
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[6] ^ cst10
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[0] ^ cst11
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[8] ^ cst3
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[11] ^ cst0
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[3] ^ cst8
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[10] ^ cst6
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[12] ^ cst2
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[4] ^ cst13
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[7] ^ cst5
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[15] ^ cst14
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[1] ^ cst9
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[14] ^ cst15
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[9] ^ cst1
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[5] ^ cst7
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[13] ^ cst4
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 7.
		v0 += m[12] ^ cst5
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[1] ^ cst15
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[14] ^ cst13
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[4] ^ cst10
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[13] ^ cst14
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[10] ^ cst4
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[15] ^ cst1
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[5] ^ cst12
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[0] ^ cst7
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[6] ^ cst3
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[9] ^ cst2
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[8] ^ cst11
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[2] ^ cst9
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[11] ^ cst8
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[3] ^ cst6
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[7] ^ cst0
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 8.
		v0 += m[13] ^ cst11
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[7] ^ cst14
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[12] ^ cst1
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[3] ^ cst9
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[1] ^ cst12
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[9] ^ cst3
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[14] ^ cst7
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[11] ^ cst13
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[5] ^ cst0
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[15] ^ cst4
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[8] ^ cst6
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[2] ^ cst10
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[6] ^ cst8
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[10] ^ cst2
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[4] ^ cst15
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[0] ^ cst5
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 9.
		v0 += m[6] ^ cst15
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[14] ^ cst9
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[11] ^ cst3
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[0] ^ cst8
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[3] ^ cst11
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[8] ^ cst0
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[9] ^ cst14
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[15] ^ cst6
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[12] ^ cst2
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[13] ^ cst7
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[1] ^ cst4
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[10] ^ cst5
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[4] ^ cst1
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[5] ^ cst10
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[7] ^ cst13
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[2] ^ cst12
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 10.
		v0 += m[10] ^ cst2
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[8] ^ cst4
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[7] ^ cst6
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[1] ^ cst5
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[6] ^ cst7
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[5] ^ cst1
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[4] ^ cst8
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[2] ^ cst10
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[15] ^ cst11
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[9] ^ cst14
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[3] ^ cst12
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[13] ^ cst0
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[12] ^ cst3
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[0] ^ cst13
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[14] ^ cst9
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[11] ^ cst15
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 11.
		v0 += m[0] ^ cst1
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[2] ^ cst3
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[4] ^ cst5
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[6] ^ cst7
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[5] ^ cst4
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[7] ^ cst6
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[3] ^ cst2
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[1] ^ cst0
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[8] ^ cst9
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[10] ^ cst11
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[12] ^ cst13
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[14] ^ cst15
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[13] ^ cst12
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[15] ^ cst14
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[11] ^ cst10
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[9] ^ cst8
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 12.
		v0 += m[14] ^ cst10
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[4] ^ cst8
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[9] ^ cst15
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[13] ^ cst6
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[15] ^ cst9
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[6] ^ cst13
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[8] ^ cst4
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[10] ^ cst14
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[1] ^ cst12
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[0] ^ cst2
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[11] ^ cst7
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[5] ^ cst3
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[7] ^ cst11
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[3] ^ cst5
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[2] ^ cst0
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[12] ^ cst1
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 13.
		v0 += m[11] ^ cst8
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[12] ^ cst0
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[5] ^ cst2
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[15] ^ cst13
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[2] ^ cst5
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[13] ^ cst15
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[0] ^ cst12
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[8] ^ cst11
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[10] ^ cst14
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[3] ^ cst6
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[7] ^ cst1
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[9] ^ cst4
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[1] ^ cst7
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[4] ^ cst9
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[6] ^ cst3
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[14] ^ cst10
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		// Round 14.
		v0 += m[7] ^ cst9
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-16) | v12>>16
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-12) | v4>>12
		v1 += m[3] ^ cst1
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-16) | v13>>16
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-12) | v5>>12
		v2 += m[13] ^ cst12
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-16) | v14>>16
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-12) | v6>>12
		v3 += m[11] ^ cst14
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-16) | v15>>16
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-12) | v7>>12
		v2 += m[12] ^ cst13
		v2 += v6
		v14 ^= v2
		v14 = v14<<(32-8) | v14>>8
		v10 += v14
		v6 ^= v10
		v6 = v6<<(32-7) | v6>>7
		v3 += m[14] ^ cst11
		v3 += v7
		v15 ^= v3
		v15 = v15<<(32-8) | v15>>8
		v11 += v15
		v7 ^= v11
		v7 = v7<<(32-7) | v7>>7
		v1 += m[1] ^ cst3
		v1 += v5
		v13 ^= v1
		v13 = v13<<(32-8) | v13>>8
		v9 += v13
		v5 ^= v9
		v5 = v5<<(32-7) | v5>>7
		v0 += m[9] ^ cst7
		v0 += v4
		v12 ^= v0
		v12 = v12<<(32-8) | v12>>8
		v8 += v12
		v4 ^= v8
		v4 = v4<<(32-7) | v4>>7
		v0 += m[2] ^ cst6
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-16) | v15>>16
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-12) | v5>>12
		v1 += m[5] ^ cst10
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-16) | v12>>16
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-12) | v6>>12
		v2 += m[4] ^ cst0
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-16) | v13>>16
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-12) | v7>>12
		v3 += m[15] ^ cst8
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-16) | v14>>16
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-12) | v4>>12
		v2 += m[0] ^ cst4
		v2 += v7
		v13 ^= v2
		v13 = v13<<(32-8) | v13>>8
		v8 += v13
		v7 ^= v8
		v7 = v7<<(32-7) | v7>>7
		v3 += m[8] ^ cst15
		v3 += v4
		v14 ^= v3
		v14 = v14<<(32-8) | v14>>8
		v9 += v14
		v4 ^= v9
		v4 = v4<<(32-7) | v4>>7
		v1 += m[10] ^ cst5
		v1 += v6
		v12 ^= v1
		v12 = v12<<(32-8) | v12>>8
		v11 += v12
		v6 ^= v11
		v6 = v6<<(32-7) | v6>>7
		v0 += m[6] ^ cst2
		v0 += v5
		v15 ^= v0
		v15 = v15<<(32-8) | v15>>8
		v10 += v15
		v5 ^= v10
		v5 = v5<<(32-7) | v5>>7

		h0 ^= v0 ^ v8 ^ s0
		h1 ^= v1 ^ v9 ^ s1
		h2 ^= v2 ^ v10 ^ s2
		h3 ^= v3 ^ v11 ^ s3
		h4 ^= v4 ^ v12 ^ s0
		h5 ^= v5 ^ v13 ^ s1
		h6 ^= v6 ^ v14 ^ s2
		h7 ^= v7 ^ v15 ^ s3

		p = p[BlockSize:]
	}
	d.h[0], d.h[1], d.h[2], d.h[3], d.h[4], d.h[5], d.h[6], d.h[7] = h0, h1, h2, h3, h4, h5, h6, h7
}
