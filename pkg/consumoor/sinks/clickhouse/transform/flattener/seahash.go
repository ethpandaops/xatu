package flattener

const (
	seaHashSeedA = uint64(0x16f11fe89b0d677c)
	seaHashSeedB = uint64(0xb480a793d8e6c86c)
	seaHashSeedC = uint64(0x6fe2e5aaf078ebc9)
	seaHashSeedD = uint64(0x14f994a4c5259381)
)

// SeaHash64 returns a SeaHash digest for the provided string.
//
// It mirrors the reference algorithm used by Vector's VRL `seahash(...)`
// function so hash-derived keys match existing Vector output.
func SeaHash64(s string) uint64 {
	data := []byte(s)

	a := seaHashSeedA
	b := seaHashSeedB
	c := seaHashSeedC
	d := seaHashSeedD

	for i := 0; i < len(data); i += 8 {
		end := i + 8
		if end > len(data) {
			end = len(data)
		}

		x := readSeaHashInt(data[i:end])
		next := diffuseSeaHash(a ^ x)

		a = b
		b = c
		c = d
		d = next
	}

	return diffuseSeaHash(a ^ b ^ c ^ d ^ uint64(len(data)))
}

// SeaHashInt64 returns SeaHash output cast to int64 (two's complement),
// matching how Int64 ClickHouse columns represent hashes.
func SeaHashInt64(s string) int64 {
	return int64(SeaHash64(s)) //nolint:gosec // intentional two's complement cast for hash
}

func readSeaHashInt(b []byte) uint64 {
	var x uint64

	for i := len(b) - 1; i >= 0; i-- {
		x <<= 8
		x |= uint64(b[i])
	}

	return x
}

func diffuseSeaHash(x uint64) uint64 {
	x *= 0x6eed0e9da4d94a4f

	a := x >> 32
	b := x >> 60

	x ^= a >> b
	x *= 0x6eed0e9da4d94a4f

	return x
}
