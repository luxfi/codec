// Copyright (C) 2019-2025, Lux Industries, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package linearcodec

import (
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/luxfi/codec"
)

const (
	// DefaultMaxSliceLen is the default max slice length (2 MiB to allow large blocks)
	DefaultMaxSliceLen = 2 * 1024 * 1024
)

var (
	ErrCantRegisterType = errors.New("can't register type")
	ErrTypeNotFound     = errors.New("type not found")
)

// Codec is a linear codec for serialization
type Codec struct {
	lock        sync.RWMutex
	maxSliceLen int
	nextTypeID  uint32
	typeIDToIdx map[reflect.Type]uint32
	idxToType   map[uint32]reflect.Type
}

// New returns a new linear codec with the default max slice length
func New(maxSliceLen int) *Codec {
	return &Codec{
		maxSliceLen: maxSliceLen,
		nextTypeID:  0,
		typeIDToIdx: make(map[reflect.Type]uint32),
		idxToType:   make(map[uint32]reflect.Type),
	}
}

// NewDefault returns a new linear codec with the default max slice length
func NewDefault() *Codec {
	return New(DefaultMaxSliceLen)
}

// SkipRegistrations skips the next n type IDs (for backwards compatibility)
func (c *Codec) SkipRegistrations(num int) {
	c.lock.Lock()
	c.nextTypeID += uint32(num)
	c.lock.Unlock()
}

// RegisterType registers a type for serialization
func (c *Codec) RegisterType(val interface{}) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	t := reflect.TypeOf(val)
	if _, exists := c.typeIDToIdx[t]; exists {
		return fmt.Errorf("%w: %v already registered", ErrCantRegisterType, t)
	}

	typeID := c.nextTypeID
	c.typeIDToIdx[t] = typeID
	c.idxToType[typeID] = t
	c.nextTypeID++
	return nil
}

// MarshalInto marshals the value into the packer
func (c *Codec) MarshalInto(val interface{}, p *codec.Packer) error {
	return c.marshal(reflect.ValueOf(val), p)
}

// UnmarshalFrom unmarshals from the packer into the value
func (c *Codec) UnmarshalFrom(p *codec.Packer, val interface{}) error {
	rv := reflect.ValueOf(val)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("%w: need pointer to unmarshal", codec.ErrUnsupportedType)
	}
	return c.unmarshal(p, rv.Elem())
}

// Size returns the serialized size of the value
func (c *Codec) Size(val interface{}) (int, error) {
	return c.size(reflect.ValueOf(val))
}

func (c *Codec) marshal(rv reflect.Value, p *codec.Packer) error {
	if p.Errored() {
		return p.Err
	}

	switch rv.Kind() {
	case reflect.Bool:
		p.PackBool(rv.Bool())
	case reflect.Uint8:
		p.PackByte(byte(rv.Uint()))
	case reflect.Uint16:
		p.PackShort(uint16(rv.Uint()))
	case reflect.Uint32:
		p.PackInt(uint32(rv.Uint()))
	case reflect.Uint64:
		p.PackLong(rv.Uint())
	case reflect.Int8:
		p.PackByte(byte(rv.Int()))
	case reflect.Int16:
		p.PackShort(uint16(rv.Int()))
	case reflect.Int32:
		p.PackInt(uint32(rv.Int()))
	case reflect.Int64:
		p.PackLong(uint64(rv.Int()))
	case reflect.String:
		p.PackStr(rv.String())
	case reflect.Slice:
		if rv.IsNil() {
			p.PackInt(0)
			return p.Err
		}
		if rv.Len() > c.maxSliceLen {
			return codec.ErrMaxSliceLenExceeded
		}
		p.PackInt(uint32(rv.Len()))
		for i := 0; i < rv.Len(); i++ {
			if err := c.marshal(rv.Index(i), p); err != nil {
				return err
			}
		}
	case reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if err := c.marshal(rv.Index(i), p); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			if err := c.marshal(rv.Field(i), p); err != nil {
				return err
			}
		}
	case reflect.Ptr:
		if rv.IsNil() {
			return codec.ErrMarshalZeroLength
		}
		return c.marshal(rv.Elem(), p)
	case reflect.Interface:
		if rv.IsNil() {
			return codec.ErrMarshalZeroLength
		}
		c.lock.RLock()
		idx, ok := c.typeIDToIdx[rv.Elem().Type()]
		c.lock.RUnlock()
		if !ok {
			return fmt.Errorf("%w: %v", ErrTypeNotFound, rv.Elem().Type())
		}
		p.PackInt(idx)
		return c.marshal(rv.Elem(), p)
	default:
		return fmt.Errorf("%w: %v", codec.ErrUnsupportedType, rv.Kind())
	}
	return p.Err
}

func (c *Codec) unmarshal(p *codec.Packer, rv reflect.Value) error {
	if p.Errored() {
		return p.Err
	}

	switch rv.Kind() {
	case reflect.Bool:
		rv.SetBool(p.UnpackBool())
	case reflect.Uint8:
		rv.SetUint(uint64(p.UnpackByte()))
	case reflect.Uint16:
		rv.SetUint(uint64(p.UnpackShort()))
	case reflect.Uint32:
		rv.SetUint(uint64(p.UnpackInt()))
	case reflect.Uint64:
		rv.SetUint(p.UnpackLong())
	case reflect.Int8:
		rv.SetInt(int64(p.UnpackByte()))
	case reflect.Int16:
		rv.SetInt(int64(p.UnpackShort()))
	case reflect.Int32:
		rv.SetInt(int64(p.UnpackInt()))
	case reflect.Int64:
		rv.SetInt(int64(p.UnpackLong()))
	case reflect.String:
		rv.SetString(p.UnpackStr())
	case reflect.Slice:
		length := int(p.UnpackInt())
		if length > c.maxSliceLen {
			return codec.ErrMaxSliceLenExceeded
		}
		slice := reflect.MakeSlice(rv.Type(), length, length)
		for i := 0; i < length; i++ {
			if err := c.unmarshal(p, slice.Index(i)); err != nil {
				return err
			}
		}
		rv.Set(slice)
	case reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			if err := c.unmarshal(p, rv.Index(i)); err != nil {
				return err
			}
		}
	case reflect.Struct:
		for i := 0; i < rv.NumField(); i++ {
			if err := c.unmarshal(p, rv.Field(i)); err != nil {
				return err
			}
		}
	case reflect.Ptr:
		elem := reflect.New(rv.Type().Elem())
		if err := c.unmarshal(p, elem.Elem()); err != nil {
			return err
		}
		rv.Set(elem)
	case reflect.Interface:
		c.lock.RLock()
		idx := p.UnpackInt()
		t, ok := c.idxToType[idx]
		if !ok {
			c.lock.RUnlock()
			return fmt.Errorf("%w: index %d", ErrTypeNotFound, idx)
		}
		c.lock.RUnlock()
		elem := reflect.New(t).Elem()
		if err := c.unmarshal(p, elem); err != nil {
			return err
		}
		rv.Set(elem)
	default:
		return fmt.Errorf("%w: %v", codec.ErrUnsupportedType, rv.Kind())
	}
	return p.Err
}

func (c *Codec) size(rv reflect.Value) (int, error) {
	switch rv.Kind() {
	case reflect.Bool, reflect.Uint8, reflect.Int8:
		return 1, nil
	case reflect.Uint16, reflect.Int16:
		return 2, nil
	case reflect.Uint32, reflect.Int32:
		return 4, nil
	case reflect.Uint64, reflect.Int64:
		return 8, nil
	case reflect.String:
		return 2 + len(rv.String()), nil
	case reflect.Slice:
		if rv.IsNil() {
			return 4, nil
		}
		size := 4
		for i := 0; i < rv.Len(); i++ {
			s, err := c.size(rv.Index(i))
			if err != nil {
				return 0, err
			}
			size += s
		}
		return size, nil
	case reflect.Array:
		size := 0
		for i := 0; i < rv.Len(); i++ {
			s, err := c.size(rv.Index(i))
			if err != nil {
				return 0, err
			}
			size += s
		}
		return size, nil
	case reflect.Struct:
		size := 0
		for i := 0; i < rv.NumField(); i++ {
			s, err := c.size(rv.Field(i))
			if err != nil {
				return 0, err
			}
			size += s
		}
		return size, nil
	case reflect.Ptr:
		if rv.IsNil() {
			return 0, codec.ErrMarshalZeroLength
		}
		return c.size(rv.Elem())
	case reflect.Interface:
		if rv.IsNil() {
			return 0, codec.ErrMarshalZeroLength
		}
		s, err := c.size(rv.Elem())
		if err != nil {
			return 0, err
		}
		return 4 + s, nil
	default:
		return 0, fmt.Errorf("%w: %v", codec.ErrUnsupportedType, rv.Kind())
	}
}
