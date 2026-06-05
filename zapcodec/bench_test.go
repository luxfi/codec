// Copyright (C) 2025-2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapcodec

import (
	"testing"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
)

// Benchmark fixtures
//
// These shapes are deliberately close to representative platformvm
// transaction payloads — many integer fields, a couple of slices, a
// fixed-width hash array, an interface dispatch. Synthetic enough to
// not pull in node deps; realistic enough that the marshal/unmarshal
// cost dominates over the test harness.

type benchTx struct {
	NetworkID uint32      `serialize:"true"`
	BlockchainID [32]byte `serialize:"true"`
	Inputs    []benchInput `serialize:"true"`
	Outputs   []benchOutput `serialize:"true"`
	Memo      []byte       `serialize:"true"`
}

type benchInput struct {
	TxID    [32]byte `serialize:"true"`
	OutIdx  uint32   `serialize:"true"`
	AssetID [32]byte `serialize:"true"`
	Amount  uint64   `serialize:"true"`
}

type benchOutput struct {
	AssetID  [32]byte `serialize:"true"`
	Amount   uint64   `serialize:"true"`
	Locktime uint64   `serialize:"true"`
	Threshold uint32  `serialize:"true"`
	Addrs    [][20]byte `serialize:"true"`
}

func benchFixture() *benchTx {
	out := func(amount uint64) benchOutput {
		return benchOutput{
			AssetID: [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
				17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
			Amount: amount, Locktime: 0, Threshold: 1,
			Addrs: [][20]byte{
				{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19,
					0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F, 0x20, 0x21, 0x22, 0x23},
			},
		}
	}
	in := func(idx uint32) benchInput {
		return benchInput{
			TxID:    [32]byte{0xCA, 0xFE, 0xBA, 0xBE, 0xDE, 0xAD, 0xBE, 0xEF},
			OutIdx:  idx,
			AssetID: [32]byte{0x42, 0x42, 0x42},
			Amount:  1_000_000_000,
		}
	}
	return &benchTx{
		NetworkID: 1,
		BlockchainID: [32]byte{0x11, 0x22, 0x33, 0x44},
		Inputs:  []benchInput{in(0), in(1), in(2), in(3)},
		Outputs: []benchOutput{out(100), out(200), out(300), out(400)},
		Memo:    []byte("benchmark transaction memo"),
	}
}

func benchManager(b *testing.B, mk func() codec.Codec) codec.Manager {
	b.Helper()
	m := codec.NewDefaultManager()
	c := mk()
	if err := m.RegisterCodec(0, c); err != nil {
		b.Fatal(err)
	}
	return m
}

// ---- Marshal ----

func BenchmarkMarshalLinear(b *testing.B) {
	m := benchManager(b, func() codec.Codec { return linearcodec.NewDefault() })
	tx := benchFixture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := m.Marshal(0, tx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMarshalZap(b *testing.B) {
	m := benchManager(b, func() codec.Codec { return NewDefault() })
	tx := benchFixture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := m.Marshal(0, tx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ---- Unmarshal ----

func BenchmarkUnmarshalLinear(b *testing.B) {
	m := benchManager(b, func() codec.Codec { return linearcodec.NewDefault() })
	tx := benchFixture()
	buf, err := m.Marshal(0, tx)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out benchTx
		if _, err := m.Unmarshal(buf, &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnmarshalZap(b *testing.B) {
	m := benchManager(b, func() codec.Codec { return NewDefault() })
	tx := benchFixture()
	buf, err := m.Marshal(0, tx)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out benchTx
		if _, err := m.Unmarshal(buf, &out); err != nil {
			b.Fatal(err)
		}
	}
}

// ---- Size ----

func BenchmarkSizeLinear(b *testing.B) {
	m := benchManager(b, func() codec.Codec { return linearcodec.NewDefault() })
	tx := benchFixture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := m.Size(0, tx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSizeZap(b *testing.B) {
	m := benchManager(b, func() codec.Codec { return NewDefault() })
	tx := benchFixture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := m.Size(0, tx); err != nil {
			b.Fatal(err)
		}
	}
}
