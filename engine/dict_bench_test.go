package engine

import (
	"fmt"
	"testing"
)

var (
	benchmarkValue Term
	benchmarkFound bool
)

func BenchmarkDictValue(b *testing.B) {
	cases := []struct {
		name string
		size int
		key  Atom
	}{
		// size 1
		{name: "size_1_hit", size: 1, key: asKey(0)},
		{name: "size_1_miss_low", size: 1, key: NewAtom("j")}, // "j" < "k000000000"
		{name: "size_1_miss_high", size: 1, key: asKey(1)},    //  "k00000001" > last

		// size 16
		{name: "size_16_hit_first", size: 16, key: asKey(0)},
		{name: "size_16_hit_mid", size: 16, key: asKey(8)},
		{name: "size_16_hit_last", size: 16, key: asKey(15)},
		{name: "size_16_miss_low", size: 16, key: NewAtom("j")}, // below "k000000000"
		{name: "size_16_miss_high", size: 16, key: asKey(16)},   // "k000000016" > last

		// size 1024
		{name: "size_1024_hit_first", size: 1024, key: asKey(0)},
		{name: "size_1024_hit_mid", size: 1024, key: asKey(512)},
		{name: "size_1024_hit_last", size: 1024, key: asKey(1023)},
		{name: "size_1024_miss_low", size: 1024, key: NewAtom("j")},
		{name: "size_1024_miss_high", size: 1024, key: asKey(1024)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			dict, err := buildSequentialDict(tc.size)
			if err != nil {
				b.Fatalf("failed to build dict: %v", err)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				benchmarkValue, benchmarkFound = dict.Value(tc.key)
			}
		})
	}
}

func buildSequentialDict(size int) (Dict, error) {
	args := make([]Term, 0, size*2+1)
	args = append(args, NewAtom("benchmark"))
	for i := 0; i < size; i++ {
		args = append(args, asKey(i), Integer(i))
	}

	return NewDict(args)
}

func asKey(i int) Atom {
	return NewAtom(fmt.Sprintf("k%09d", i))
}
