package nordlead3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/dgryski/go-bitstream"
)

const (
	sysexStart = 0xF0
	sysexEnd   = 0xF7
)

const (
	programFromSlot       = 0x20
	programFromMemory     = 0x21
	performanceFromSlot   = 0x28
	performanceFromMemory = 0x29
)

const (
	categoryOffset  = 22
	versionOffset   = 38
	patchdataOffset = 40
)

const (
	spareHeaderLength          = 15
	performanceBitstreamLength = 859
	programBitstreamLength     = 191
)

var sysexHeader = []byte{0xF0, 0x33, 0x7F, 0x09}

type sysex struct {
	rawSysex         []byte
	decodedBitstream []byte
}

type sysexable interface {
	sysexData() (*[]byte, error)
	sysexType() uint8
	sysexName() []byte
	sysexCategory() uint8
	sysexVersion() []byte
}

func (s *sysex) bank() int {
	return int(s.rawSysex[4])
}

func (s *sysex) decodeBitstream() {
	s.decodedBitstream = unpackSysex(s.rawBitstream())
}

func (s *sysex) rawBitstream() []byte {
	return s.rawSysex[patchdataOffset:]
}

func (s *sysex) location() int {
	return int(s.rawSysex[5])
}

func (s *sysex) checksum() uint8 {
	return s.decodedBitstream[len(s.decodedBitstream)-1]
}

func (s *sysex) messageType() uint8 {
	return s.rawSysex[3]
}

func (s *sysex) category() uint8 {
	return s.rawSysex[categoryOffset]
}

func (s *sysex) name() []byte {
	return s.rawSysex[6:22]
}

func (s *sysex) nameAsArray() [16]byte {
	var name [16]byte
	for i, char := range s.name() {
		name[i] = char
	}
	return name
}

func (s *sysex) printableName() string {
	return fmt.Sprintf("%-16s", strings.TrimRight(string(s.name()), "\x00"))
}

func (s *sysex) printableType() string {
	switch s.messageType() {
	case programFromSlot, programFromMemory:
		return "Program"
	case performanceFromSlot, performanceFromMemory:
		return "Performance"
	default:
		return "Unknown"
	}
}

func (s *sysex) valid() (bool, error) {
	var errStrs []string

	// Verify message type and expected length
	switch s.messageType() {
	case programFromSlot, programFromMemory:
		if len(s.decodedBitstream) != programBitstreamLength {
			errStrs = append(errStrs, fmt.Sprintf("Error parsing %s (%v:%03d %q): data invalid!", s.printableType(), s.bank(), s.location(), s.printableName()))
		}
	case performanceFromSlot, performanceFromMemory:
		if len(s.decodedBitstream) != performanceBitstreamLength {
			errStrs = append(errStrs, fmt.Sprintf("Error parsing %s (%v:%03d %q): data invalid!", s.printableType(), s.bank(), s.location(), s.printableName()))
		}
	default:
		errStrs = append(errStrs, fmt.Sprintf("Unknown type %x (%d)", s.messageType(), s.messageType()))
	}

	// Compute and validate 8-bit checksum
	checksum := s.decodedBitstream[len(s.decodedBitstream)-1]
	payload := s.decodedBitstream[:len(s.decodedBitstream)-1]
	calculatedChecksum := checksum8(payload)
	if checksum != calculatedChecksum {
		errStrs = append(errStrs, fmt.Sprintf("Checksum mismatch parsing %s (%v:%03d %q): expected %x, got %x", s.printableType(), s.bank(), s.location(), s.printableName(), checksum, calculatedChecksum))
	}

	// Handle return values
	if len(errStrs) == 0 {
		return true, nil
	} else {
		return false, errors.New(strings.Join(errStrs, " "))
	}
}

func (s *sysex) version() float64 {
	return float64(uint16(s.rawSysex[versionOffset])<<8+uint16(s.rawSysex[versionOffset+1])) / 100.0
}

func ParseSysex(rawSysex []byte) (*sysex, error) {
	// Strip leading F0 and trailing F7, if present
	if rawSysex[0] == 0xF0 {
		rawSysex = rawSysex[1:]
	}
	if rawSysex[len(rawSysex)-1] == 0xF7 {
		rawSysex = rawSysex[:len(rawSysex)-1]
	}

	s := sysex{rawSysex: rawSysex}
	s.decodeBitstream()

	_, err := s.valid()

	return &s, err
}

// MIDI 8-bit to bitstream decoding
// Every byte of the MIDI stream is actually only 7 bits of the payload bitstream
// so we need to drop a bit every byte and re-concatenate the bits
func unpackSysex(payload []byte) []byte {
	buf := bytes.NewBuffer(nil)
	reader := bitstream.NewReader(bytes.NewReader(payload))
	writer := bitstream.NewWriter(buf)
	i := 0

	for {
		bit, err := reader.ReadBit()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(fmt.Sprintf("GetBit returned error err %v", err.Error()))
		}
		if i%8 == 0 {
			// skip
		} else {
			err = writer.WriteBit(bit)
			if err == nil {
				// skip
			} else {
				panic(fmt.Sprintf("Error writing bit: %v", err.Error()))
			}
		}
		i++
	}

	return buf.Bytes()
}

// Encodes 8-bit binary data as bytes with 7 bits of data
// in the LSB and the MSB set to 0. For transmission over sysex.
func packSysex(payload []byte) []byte {
	buf := bytes.NewBuffer(nil)
	reader := bitstream.NewReader(bytes.NewReader(payload))
	writer := bitstream.NewWriter(buf)

	for {
		var err error
		var bit bitstream.Bit

		writer.WriteBit(bitstream.Zero)
		for i := 0; i < 7; i++ {
			bit, err = reader.ReadBit()
			if err != nil && err != io.EOF {
				panic(err)
			}
			if err == io.EOF {
				break
			}
			writer.WriteBit(bit)
		}
		if err == io.EOF {
			break
		}
	}
	writer.Flush(bitstream.Zero)
	return buf.Bytes()
}

// Returns the given object as a complete sysex chunk, including F0/F7 terminators
func toSysex(obj sysexable, ref patchRef) (*[]byte, error) {
	buffer := bytes.NewBuffer(nil)

	buffer.Write(sysexHeader)
	buffer.Write([]byte{obj.sysexType(), uint8(ref.bank()), uint8(ref.location())})
	buffer.Write(obj.sysexName())
	buffer.WriteByte(obj.sysexCategory())
	buffer.Write((*new([spareHeaderLength]byte))[:])
	buffer.Write(obj.sysexVersion())

	payload, err := obj.sysexData()
	if err != nil {
		return nil, err
	}
	buffer.Write(*payload)

	buffer.WriteByte(0xF7)

	sysex := buffer.Bytes()
	return &sysex, nil
}
