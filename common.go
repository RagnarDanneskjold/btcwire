// Copyright (c) 2013-2014 Conformal Systems LLC.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcwire

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"github.com/conformal/fastsha256"
	"io"
	"math"
)

// Maximum payload size for a variable length integer.
const MaxVarIntPayload = 9

// readElement reads the next sequence of bytes from r using little endian
// depending on the concrete type of element pointed to.
func readElement(r io.Reader, element interface{}) error {
	var scratch [8]byte

	// Attempt to read the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case *int32:
		b := scratch[0:4]
		_, err := io.ReadFull(r, b)
		if err != nil {
			return err
		}
		*e = int32(binary.LittleEndian.Uint32(b))
		return nil

	case *uint32:
		b := scratch[0:4]
		_, err := io.ReadFull(r, b)
		if err != nil {
			return err
		}
		*e = binary.LittleEndian.Uint32(b)
		return nil

	case *int64:
		b := scratch[0:8]
		_, err := io.ReadFull(r, b)
		if err != nil {
			return err
		}
		*e = int64(binary.LittleEndian.Uint64(b))
		return nil

	case *uint64:
		b := scratch[0:8]
		_, err := io.ReadFull(r, b)
		if err != nil {
			return err
		}
		*e = binary.LittleEndian.Uint64(b)
		return nil

	// Message header checksum.
	case *[4]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case *[commandSize]uint8:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case *[16]byte:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *ShaHash:
		_, err := io.ReadFull(r, e[:])
		if err != nil {
			return err
		}
		return nil

	case *ServiceFlag:
		b := scratch[0:8]
		_, err := io.ReadFull(r, b)
		if err != nil {
			return err
		}
		*e = ServiceFlag(binary.LittleEndian.Uint64(b))
		return nil

	case *InvType:
		b := scratch[0:4]
		_, err := io.ReadFull(r, b)
		if err != nil {
			return err
		}
		*e = InvType(binary.LittleEndian.Uint32(b))
		return nil

	case *BitcoinNet:
		b := scratch[0:4]
		_, err := io.ReadFull(r, b)
		if err != nil {
			return err
		}
		*e = BitcoinNet(binary.LittleEndian.Uint32(b))
		return nil
	}

	// Fall back to the slower binary.Read if a fast path was not available
	// above.
	return binary.Read(r, binary.LittleEndian, element)
}

// readElements reads multiple items from r.  It is equivalent to multiple
// calls to readElement.
func readElements(r io.Reader, elements ...interface{}) error {
	for _, element := range elements {
		err := readElement(r, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// writeElement writes the little endian representation of element to w.
func writeElement(w io.Writer, element interface{}) error {
	var scratch [8]byte

	// Attempt to write the element based on the concrete type via fast
	// type assertions first.
	switch e := element.(type) {
	case int32:
		b := scratch[0:4]
		binary.LittleEndian.PutUint32(b, uint32(e))
		_, err := w.Write(b)
		if err != nil {
			return err
		}
		return nil

	case uint32:
		b := scratch[0:4]
		binary.LittleEndian.PutUint32(b, e)
		_, err := w.Write(b)
		if err != nil {
			return err
		}
		return nil

	case int64:
		b := scratch[0:8]
		binary.LittleEndian.PutUint64(b, uint64(e))
		_, err := w.Write(b)
		if err != nil {
			return err
		}
		return nil

	case uint64:
		b := scratch[0:8]
		binary.LittleEndian.PutUint64(b, e)
		_, err := w.Write(b)
		if err != nil {
			return err
		}
		return nil

	// Message header checksum.
	case [4]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// Message header command.
	case [commandSize]uint8:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	// IP address.
	case [16]byte:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case *ShaHash:
		_, err := w.Write(e[:])
		if err != nil {
			return err
		}
		return nil

	case ServiceFlag:
		b := scratch[0:8]
		binary.LittleEndian.PutUint64(b, uint64(e))
		_, err := w.Write(b)
		if err != nil {
			return err
		}
		return nil

	case InvType:
		b := scratch[0:4]
		binary.LittleEndian.PutUint32(b, uint32(e))
		_, err := w.Write(b)
		if err != nil {
			return err
		}
		return nil

	case BitcoinNet:
		b := scratch[0:4]
		binary.LittleEndian.PutUint32(b, uint32(e))
		_, err := w.Write(b)
		if err != nil {
			return err
		}
		return nil
	}

	// Fall back to the slower binary.Write if a fast path was not available
	// above.
	return binary.Write(w, binary.LittleEndian, element)
}

// writeElements writes multiple items to w.  It is equivalent to multiple
// calls to writeElement.
func writeElements(w io.Writer, elements ...interface{}) error {
	for _, element := range elements {
		err := writeElement(w, element)
		if err != nil {
			return err
		}
	}
	return nil
}

// readVarInt reads a variable length integer from r and returns it as a uint64.
func readVarInt(r io.Reader, pver uint32) (uint64, error) {
	b := make([]byte, 8)
	_, err := io.ReadFull(r, b[0:1])
	if err != nil {
		return 0, err
	}

	var rv uint64
	discriminant := uint8(b[0])
	switch discriminant {
	case 0xff:
		_, err := io.ReadFull(r, b)
		if err != nil {
			return 0, err
		}
		rv = binary.LittleEndian.Uint64(b)

	case 0xfe:
		_, err := io.ReadFull(r, b[0:4])
		if err != nil {
			return 0, err
		}
		rv = uint64(binary.LittleEndian.Uint32(b))

	case 0xfd:
		_, err := io.ReadFull(r, b[0:2])
		if err != nil {
			return 0, err
		}
		rv = uint64(binary.LittleEndian.Uint16(b))

	default:
		rv = uint64(discriminant)
	}

	return rv, nil
}

// writeVarInt serializes val to w using a variable number of bytes depending
// on its value.
func writeVarInt(w io.Writer, pver uint32, val uint64) error {
	if val < 0xfd {
		_, err := w.Write([]byte{uint8(val)})
		return err
	}

	if val <= math.MaxUint16 {
		buf := make([]byte, 3)
		buf[0] = 0xfd
		binary.LittleEndian.PutUint16(buf[1:], uint16(val))
		_, err := w.Write(buf)
		return err
	}

	if val <= math.MaxUint32 {
		buf := make([]byte, 5)
		buf[0] = 0xfe
		binary.LittleEndian.PutUint32(buf[1:], uint32(val))
		_, err := w.Write(buf)
		return err
	}

	buf := make([]byte, 9)
	buf[0] = 0xff
	binary.LittleEndian.PutUint64(buf[1:], val)
	_, err := w.Write(buf)
	return err
}

// VarIntSerializeSize returns the number of bytes it would take to serialize
// val as a variable length integer.
func VarIntSerializeSize(val uint64) int {
	// The value is small enough to be represented by itself, so it's
	// just 1 byte.
	if val < 0xfd {
		return 1
	}

	// Discriminant 1 byte plus 2 bytes for the uint16.
	if val <= math.MaxUint16 {
		return 3
	}

	// Discriminant 1 byte plus 4 bytes for the uint32.
	if val <= math.MaxUint32 {
		return 5
	}

	// Discriminant 1 byte plus 8 bytes for the uint64.
	return 9
}

// readVarString reads a variable length string from r and returns it as a Go
// string.  A varString is encoded as a varInt containing the length of the
// string, and the bytes that represent the string itself.  An error is returned
// if the length is greater than the maximum block payload size, since it would
// not be possible to put a varString of that size into a block anyways and it
// also helps protect against memory exhuastion attacks and forced panics
// through malformed messages.
func readVarString(r io.Reader, pver uint32) (string, error) {
	count, err := readVarInt(r, pver)
	if err != nil {
		return "", err
	}

	// Prevent variable length strings that are larger than the maximum
	// message size.  It would be possible to cause memory exhaustion and
	// panics without a sane upper bound on this count.
	if count > maxMessagePayload {
		str := fmt.Sprintf("variable length string is too long "+
			"[count %d, max %d]", count, maxMessagePayload)
		return "", messageError("readVarString", str)
	}

	buf := make([]byte, count)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// writeVarString serializes str to w as a varInt containing the length of the
// string followed by the bytes that represent the string itself.
func writeVarString(w io.Writer, pver uint32, str string) error {
	err := writeVarInt(w, pver, uint64(len(str)))
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(str))
	if err != nil {
		return err
	}
	return nil
}

// randomUint64 returns a cryptographically random uint64 value.  This
// unexported version takes a reader primarily to ensure the error paths
// can be properly tested by passing a fake reader in the tests.
func randomUint64(r io.Reader) (uint64, error) {
	b := make([]byte, 8)
	n, err := r.Read(b)
	if n != len(b) {
		return 0, io.ErrShortBuffer
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(b), nil
}

// RandomUint64 returns a cryptographically random uint64 value.
func RandomUint64() (uint64, error) {
	return randomUint64(rand.Reader)
}

// DoubleSha256 calculates sha256(sha256(b)) and returns the resulting bytes.
func DoubleSha256(b []byte) []byte {
	hasher := fastsha256.New()
	hasher.Write(b)
	sum := hasher.Sum(nil)
	hasher.Reset()
	hasher.Write(sum)
	return hasher.Sum(nil)
}
