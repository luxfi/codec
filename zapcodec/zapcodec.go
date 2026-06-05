// Copyright (C) 2025-2026, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package zapcodec is a drop-in replacement for linearcodec that emits
// the ZAP-native little-endian wire layout. It implements the same
// codec.GeneralCodec (= codec.Registry + codec.Codec) surface so a
// codec.Manager can host both side by side.
//
// Wire-format delta vs linearcodec:
//
//   - All multi-byte integers are little-endian. (linearcodec is big-
//     endian.) x86_64 and arm64 hardware is LE-native, so LE writes
//     map to single MOV instructions where BE writes need BSWAP.
//   - Interface type-id prefixes are uint32 LE (linearcodec also uses
//     uint32 but in BE).
//   - String length prefix is uint16 LE (linearcodec uses uint16 BE).
//   - All else is byte-equivalent to linearcodec: slice/map length
//     prefixes are uint32, bool is a single byte, struct fields are
//     emitted in serialize-tag order.
//
// What is NOT in this codec:
//
//   - No ZAP wire header. The codec.Manager prepends a uint16 codec
//     version (PackShort) and that BigEndian-encoded short is the
//     only BE bytes in a zapcodec-encoded blob. Everything the codec
//     itself writes is little-endian.
//   - No deferred-write/zero-copy ZAP Object/List machinery. That's a
//     separate concern handled by zap_native at the platformvm tx
//     layer. zapcodec is the generic reflection-driven codec that
//     hosts all platformvm tx types — same role linearcodec plays
//     today.
package zapcodec

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/luxfi/codec"
	"github.com/luxfi/codec/reflectcodec"
	"github.com/luxfi/codec/wrappers"
	"github.com/luxfi/container/bimap"
)

// DefaultTagName is the struct tag this codec honours (same as
// linearcodec: "serialize").
const DefaultTagName = reflectcodec.DefaultTagName

var (
	_ Codec              = (*linearCodec)(nil)
	_ codec.Codec        = (*linearCodec)(nil)
	_ codec.Registry     = (*linearCodec)(nil)
	_ codec.GeneralCodec = (*linearCodec)(nil)
)

// Codec is the zapcodec local interface — codec.GeneralCodec extended
// with SkipRegistrations so call sites can preserve linearcodec slot
// layouts during a coexistence window.
type Codec interface {
	codec.Registry
	codec.Codec
	SkipRegistrations(int)
}

// linearCodec is the concrete impl. Named "linearCodec" (lower-case L)
// for direct symmetry with linearcodec.linearCodec — when a future PR
// flips imports from linearcodec to zapcodec the type machinery stays
// readable.
type linearCodec struct {
	reflective *reflectiveCodec

	lock            sync.RWMutex
	nextTypeID      uint32
	registeredTypes *bimap.BiMap[uint32, reflect.Type]
}

// New returns a zapcodec instance that honours the given struct tag
// names. Concurrency-safe.
func New(tagNames []string) Codec {
	c := &linearCodec{
		nextTypeID:      0,
		registeredTypes: bimap.New[uint32, reflect.Type](),
	}
	c.reflective = newReflective(c, tagNames)
	return c
}

// NewDefault is the standard constructor — uses the "serialize" tag.
func NewDefault() Codec {
	return New([]string{DefaultTagName})
}

// SkipRegistrations bumps the next-type-id counter by n. Use for
// matching legacy linearcodec slot layouts when migrating.
func (c *linearCodec) SkipRegistrations(n int) {
	c.lock.Lock()
	c.nextTypeID += uint32(n)
	c.lock.Unlock()
}

// RegisterType registers val so it may be unmarshalled into an
// interface field. The type-id assigned is the current value of the
// next-type-id counter.
func (c *linearCodec) RegisterType(val interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	t := reflect.TypeOf(val)
	if c.registeredTypes.HasValue(t) {
		return fmt.Errorf("%w: %v", codec.ErrDuplicateType, t)
	}
	c.registeredTypes.Put(c.nextTypeID, t)
	c.nextTypeID++
	return nil
}

// PrefixSize is the size of an interface type-id prefix. Always 4
// bytes (uint32). Implements reflectcodec.TypeCodec-style API for the
// reflective codec.
func (*linearCodec) PrefixSize(reflect.Type) int { return intLen }

// PackPrefix writes the type-id prefix for valueType into p.
func (c *linearCodec) PackPrefix(p *packer, valueType reflect.Type) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	id, ok := c.registeredTypes.GetKey(valueType)
	if !ok {
		return fmt.Errorf("can't marshal unregistered type %q", valueType)
	}
	p.PackInt(id)
	return p.err
}

// UnpackPrefix reads a type-id prefix and returns a new value of the
// concrete implementing type.
func (c *linearCodec) UnpackPrefix(p *packer, valueType reflect.Type) (reflect.Value, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	id := p.UnpackInt()
	if p.err != nil {
		return reflect.Value{}, fmt.Errorf("couldn't unmarshal interface: %w", p.err)
	}
	implT, ok := c.registeredTypes.GetValue(id)
	if !ok {
		return reflect.Value{}, fmt.Errorf("couldn't unmarshal interface: unknown type ID %d", id)
	}
	if !implT.Implements(valueType) {
		return reflect.Value{}, fmt.Errorf("couldn't unmarshal interface: %s %w %s",
			implT, codec.ErrDoesNotImplementInterface, valueType)
	}
	return reflect.New(implT).Elem(), nil
}

// MarshalInto satisfies codec.Codec. It writes value into the supplied
// wrappers.Packer. The Packer carries a BigEndian view internally; we
// only use its Bytes/Offset/MaxSize fields and write through our own
// little-endian packer over the same backing buffer.
//
// The local zapcodec packer is stack-allocated and aliases p's buffer
// directly — there's no per-Marshal heap alloc for the packer itself.
func (c *linearCodec) MarshalInto(value interface{}, p *wrappers.Packer) error {
	if value == nil {
		return codec.ErrMarshalNil
	}
	zp := packer{
		err:     p.Err,
		maxSize: p.MaxSize,
		bytes:   p.Bytes,
		offset:  p.Offset,
	}
	err := c.reflective.Marshal(value, &zp)
	p.Bytes = zp.bytes
	p.Offset = zp.offset
	if p.Err == nil {
		p.Err = zp.err
	}
	if err != nil {
		return err
	}
	return p.Err
}

// UnmarshalFrom satisfies codec.Codec.
func (c *linearCodec) UnmarshalFrom(p *wrappers.Packer, dest interface{}) error {
	zp := packer{
		err:     p.Err,
		maxSize: p.MaxSize,
		bytes:   p.Bytes,
		offset:  p.Offset,
	}
	err := c.reflective.Unmarshal(&zp, dest)
	p.Bytes = zp.bytes
	p.Offset = zp.offset
	if p.Err == nil {
		p.Err = zp.err
	}
	if err != nil {
		return err
	}
	return p.Err
}

// Size returns the size of value as it would be marshaled. Does NOT
// include any codec.Manager-prepended version prefix.
func (c *linearCodec) Size(value interface{}) (int, error) {
	return c.reflective.Size(value)
}

