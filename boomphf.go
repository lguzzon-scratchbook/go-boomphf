package boomphf

// Fast and scalable minimal perfect hashing for massive key sets
// https://arxiv.org/abs/1702.03154

type H struct {
	b     []bitvector
	ranks [][]uint64
}

func New(gamma float64, keys []uint64) *H {

	var h H

	var level uint32

	size := uint32(gamma * float64(len(keys)))
	size = (size + 63) &^ 63
	A := newbv(size)
	collide := newbv(size)

	var redo []uint64

	for len(keys) > 0 {
		for _, v := range keys {
			hash := xorshiftMult64(v)
			hash32 := uint32(hash) + level*uint32(hash>>32)
			idx := hash32 % size

			if collide.get(idx) == 1 {
				continue
			}

			if A.get(idx) == 1 {
				A.clear(idx)
				collide.set(idx)
				continue
			}

			A.set(idx)
		}

		bv := newbv(size)
		for _, v := range keys {
			hash := xorshiftMult64(v)
			hash32 := uint32(hash) + level*uint32(hash>>32)
			idx := hash32 % size
			if collide.get(idx) == 1 {
				redo = append(redo, v)
				continue
			}

			bv.set(idx)
		}
		h.b = append(h.b, bv)

		keys = redo
		redo = redo[:0] // tricky, sharing space with `keys`
		size = uint32(gamma * float64(len(keys)))
		size = (size + 63) &^ 63
		A.reset()
		collide.reset()
		level++
	}

	h.computeRanks()

	return &h
}

func (h *H) computeRanks() {
	var pop uint64
	for _, bv := range h.b {

		r := make([]uint64, 0, 1+(len(bv)/8))

		for i, v := range bv {
			if i%8 == 0 {
				r = append(r, pop)
			}
			pop += popcnt(v)
		}
		h.ranks = append(h.ranks, r)
	}
}

func (h *H) Query(k uint64) uint64 {

	hash := xorshiftMult64(k)
	h1 := uint32(hash)
	h2 := uint32(hash >> 32)

	for i, bv := range h.b {
		hash32 := h1 + uint32(i)*h2
		idx := hash32 % (uint32(len(bv)) * 64)

		if bv.get(idx) == 0 {
			continue
		}

		rank := h.ranks[i][idx/512]

		for j := (idx / 64) &^ 7; j < idx/64; j++ {
			rank += popcnt(bv[j])
		}

		w := bv[idx/64]

		rank += popcnt(w << (64 - (idx % 64)))

		return rank + 1
	}

	return 0
}

// 64-bit xorshift multiply rng from http://vigna.di.unimi.it/ftp/papers/xorshift.pdf
func xorshiftMult64(x uint64) uint64 {
	x ^= x >> 12 // a
	x ^= x << 25 // b
	x ^= x >> 27 // c
	return x * 2685821657736338717
}

type bitvector []uint64

func newbv(size uint32) bitvector {
	return make([]uint64, uint(size+63)/64)
}

// get bit 'bit' in the bitvector d
func (b bitvector) get(bit uint32) uint {
	shift := bit % 64
	bb := b[bit/64]
	bb &= (1 << shift)

	return uint(bb >> shift)
}

// set bit 'bit' in the bitvector d
func (b bitvector) set(bit uint32) {
	b[bit/64] |= (1 << (bit % 64))
}

// set bit 'bit' in the bitvector d
func (b bitvector) clear(bit uint32) {
	b[bit/64] &= ^(1 << (bit % 64))
}

func (b bitvector) reset() {
	for i := range b {
		b[i] = 0
	}
}

func popcnt(x uint64) uint64 {
	// bit population count, see
	// http://graphics.stanford.edu/~seander/bithacks.html#CountBitsSetParallel
	x -= (x >> 1) & 0x5555555555555555
	x = (x>>2)&0x3333333333333333 + x&0x3333333333333333
	x += x >> 4
	x &= 0x0f0f0f0f0f0f0f0f
	x *= 0x0101010101010101
	return x >> 56
}
