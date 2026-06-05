// Copyright (C) 2025-2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package zapcodec

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/linearcodec"
)

// TestWireSizeParity asserts that for every fixture used in the
// benchmarks, the zapcodec wire output is byte-for-byte the same
// LENGTH as the linearcodec output. The contents differ (endianness),
// but every length-prefix and field-byte count must match — anything
// else means the codecs disagree on the layout schema.
//
// This is the structural invariant a forward-dated wire-fork is built
// on: zapcodec is supposed to be a same-shape replacement, not a
// new schema.
func TestWireSizeParity(t *testing.T) {
	require := require.New(t)

	makeMgr := func(c codec.Codec) codec.Manager {
		m := codec.NewDefaultManager()
		require.NoError(m.RegisterCodec(0, c))
		return m
	}

	lin := makeMgr(linearcodec.NewDefault())
	zap := makeMgr(NewDefault())

	tx := benchFixture()
	linBytes, err := lin.Marshal(0, tx)
	require.NoError(err)
	zapBytes, err := zap.Marshal(0, tx)
	require.NoError(err)
	require.Equal(len(linBytes), len(zapBytes),
		"zapcodec and linearcodec must emit same-length wire (lin=%d zap=%d)",
		len(linBytes), len(zapBytes))
}
