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
	SYSEX_START = 0xF0
	SYSEX_END   = 0xF7
)

const (
	ProgramFromSlot       = 0x20
	ProgramFromMemory     = 0x21
	PerformanceFromSlot   = 0x28
	PerformanceFromMemory = 0x29
)

const (
	CategoryOffset  = 22
	VersionOffset   = 38
	PatchDataOffset = 40
)

const (
	SpareHeaderLength          = 15
	PerformanceBitstreamLength = 859
	ProgramBitstreamLength     = 191
)

type Sysex struct {
	rawSysex         []byte
	decodedBitstream []byte
}

func (sysex *Sysex) bank() uint8 {
	return sysex.rawSysex[4]
}

func (sysex *Sysex) decodeBitstream() {
	payload := sysex.rawSysex[PatchDataOffset:]
	sysex.decodedBitstream = unpackSysex(payload)
}

func (sysex *Sysex) location() uint8 {
	return sysex.rawSysex[5]
}

func (sysex *Sysex) messageType() uint8 {
	return sysex.rawSysex[3]
}

func (sysex *Sysex) category() uint8 {
	return sysex.rawSysex[CategoryOffset]
}

func (sysex *Sysex) name() []byte {
	return sysex.rawSysex[6:22]
}

func (sysex *Sysex) nameAsArray() [16]byte {
	var name [16]byte
	for i, char := range sysex.name() {
		name[i] = char
	}
	return name
}

func (sysex *Sysex) printableName() string {
	return fmt.Sprintf("%-16s", strings.TrimRight(string(sysex.name()), "\x00"))
}

func (sysex *Sysex) printableType() string {
	switch sysex.messageType() {
	case ProgramFromSlot, ProgramFromMemory:
		return "Program"
	case PerformanceFromSlot, PerformanceFromMemory:
		return "Performance"
	default:
		return "Unknown"
	}
}

func (sysex *Sysex) valid() (bool, error) {
	var errStrs []string

	// Verify message type and expected length
	switch sysex.messageType() {
	case ProgramFromSlot, ProgramFromMemory:
		if len(sysex.decodedBitstream) != ProgramBitstreamLength {
			errStrs = append(errStrs, fmt.Sprintf("Error parsing %s (%v:%03d %q): data invalid!", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName()))
		}
	case PerformanceFromSlot, PerformanceFromMemory:
		if len(sysex.decodedBitstream) != PerformanceBitstreamLength {
			errStrs = append(errStrs, fmt.Sprintf("Error parsing %s (%v:%03d %q): data invalid!", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName()))
		}
	default:
		errStrs = append(errStrs, fmt.Sprintf("Unknown type %x (%d)", sysex.messageType(), sysex.messageType()))
	}

	// Compute and validate 8-bit checksum
	checksum := sysex.decodedBitstream[len(sysex.decodedBitstream)-1]
	payload := sysex.decodedBitstream[:len(sysex.decodedBitstream)-1]
	calculatedChecksum := checksum8(payload)
	if checksum != calculatedChecksum {
		errStrs = append(errStrs, fmt.Sprintf("Checksum mismatch parsing %s (%v:%03d %q): expected %x, got %x", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), checksum, calculatedChecksum))
	}

	// Handle return values
	if len(errStrs) == 0 {
		return true, nil
	} else {
		return false, errors.New(strings.Join(errStrs, " "))
	}
}

func (sysex *Sysex) version() float64 {
	return float64(uint16(sysex.rawSysex[VersionOffset])<<8+uint16(sysex.rawSysex[VersionOffset+1])) / 100.0
}

func ParseSysex(rawSysex []byte) (*Sysex, error) {
	var sysex Sysex

	// Strip leading F0 and trailing F7, if present
	if rawSysex[0] == 0xF0 {
		rawSysex = rawSysex[1:]
	}
	if rawSysex[len(rawSysex)-1] == 0xF7 {
		rawSysex = rawSysex[:len(rawSysex)-1]
	}

	sysex = Sysex{rawSysex: rawSysex}
	sysex.decodeBitstream()

	_, err := sysex.valid()

	return &sysex, err
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
		bits, err := reader.ReadBits(7)
		if err != nil && err != io.EOF {
			panic(err)
		}
		writer.WriteBits(bits, 8)
		if err == io.EOF {
			break
		}
	}

	return buf.Bytes()
}
