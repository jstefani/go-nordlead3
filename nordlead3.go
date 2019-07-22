package nordlead3

/*
TODO:

 - Get rid of programbank/performancebank abstractions as I don't think they're useful
 - Add output serializers (back to Sysex) and test roundtrip read -> output for equality
 - Write a bunch of useful tests for the core methods
 - Try to figure out how categories are implemented
 - Create useful functions for manipulating memory:
     - Swap locations
     - Rename location
     - Copy from one location to another (destination must be empty)
     - Delete a location entirely (makes destination empty)
     - Insert a location (move following locations down until an empty location is hit, or return an error if there's no room)
     - Fancy stuff: move any subset of locations (e.g. an array of tuples (bank, location)) to a consecutive block of empty destinations (e.g. (bank, location) where the first one goes)
 - Try to identify the difference between v1.18 and v1.20 Sysex and see if you can figure out where the missing arp sync settings are.
*/

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/dgryski/go-bitstream"
)

type ProgramBank struct {
	Programs [128]*ProgramLocation
}

func (bank *ProgramBank) PrintSummary(omitBlank bool) string {
	var result []string

	for location, program := range bank.Programs {
		if program != nil {
			result = append(result, fmt.Sprintf("   %3d : %+-16.16q (%1.2f)", location, program.PrintableName(), program.Version))
		} else if !omitBlank {
			result = append(result, fmt.Sprintf("   %3d : %+-16.16q", location, program.PrintableName()))
		}
	}

	return strings.Join(result, "\n")
}

type PerformanceBank struct {
	Performances [128]*PerformanceLocation
}

func (bank *PerformanceBank) PrintSummary(omitBlank bool) string {
	var result []string

	for location, performance := range bank.Performances {
		if performance != nil {
			result = append(result, fmt.Sprintf("   %3d : %16.16q (%1.2f)", location, performance.PrintableName(), performance.Version))
		} else if !omitBlank {
			result = append(result, fmt.Sprintf("   %3d : %16.16q", location, performance.PrintableName()))
		}
	}

	return strings.Join(result, "\n")
}

type ProgramLocation struct {
	Name     [16]byte
	Category uint8
	Version  float64
	Program  *Program
}

func (progLoc *ProgramLocation) PrintableName() string {
	if progLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(progLoc.Name[:]), "\x00"))
}

type PerformanceLocation struct {
	Name        [16]byte
	Category    uint8
	Version     float64
	Performance *Performance
}

func (perfLoc *PerformanceLocation) PrintableName() string {
	if perfLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(perfLoc.Name[:]), "\x00"))
}

func populateStructFromBitstream(i interface{}, data []byte) error {
	// Use reflection to get each field in the struct and it's length, then read that into it

	rt := reflect.TypeOf(i).Elem()
	rv := reflect.ValueOf(i).Elem()

	return populateReflectedStructFromBitstream(rt, rv, data)
}

func populateReflectedStructFromBitstream(rt reflect.Type, rv reflect.Value, data []byte) error {
	reader := bitstream.NewReader(bytes.NewReader(data))
	err := (error)(nil)

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i) // Type of the StructField (for reading tags)
		rf := rv.Field(i) // Value of the struct field (for setting value)

		if strLen, ok := sf.Tag.Lookup("len"); ok {
			numBitsToRead, _ := strconv.Atoi(strLen)
			switch rf.Kind() {
			case reflect.Int:
				err = readInt(rf, reader, numBitsToRead)
			case reflect.Uint:
				err = readUint(rf, reader, numBitsToRead)
			case reflect.Bool:
				err = readBool(rf, reader)
			case reflect.Array:
				size := rf.Len()

				for i := 0; i < size; i++ {
					rfi := rf.Index(i)
					err = readUint(rfi, reader, numBitsToRead)
					if err != nil {
						break
					}
				}
			case reflect.Struct:
				bytes, err := readUnaligned(reader, numBitsToRead)
				if err == nil {
					newStruct := reflect.New(sf.Type)
					// fmt.Printf("creating and populating a %q with %q. Got:\n%x\n", sf.Type, newSub.Type(), subData)
					_ = populateReflectedStructFromBitstream(newStruct.Elem().Type(), newStruct.Elem(), bytes)
					rf.Set(newStruct.Elem())
				}
			default:
				return errors.New(fmt.Sprintf("Unhandled type discovered: %v\n", rf.Kind()))
			}
		} else {
			err = errors.New(fmt.Sprintf("Length for %s not specified, not sure how to proceed!", sf.Name))
		}

		if err != nil {
			break
		}
	}

	return err
}

func bitstreamFromStruct(i interface{}) ([]byte, error) {
	rt := reflect.TypeOf(i).Elem()
	rv := reflect.ValueOf(i).Elem()

	buf := bytes.NewBuffer(nil)
	writer := bitstream.NewWriter(buf)

	err := writeBitstreamFromReflection(writer, rt, rv)
	return buf.Bytes(), err
}

func writeBitstreamFromReflection(writer *bitstream.BitWriter, rt reflect.Type, rv reflect.Value) error {
	err := (error)(nil)

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i) // Type of the StructField (for reading tags)
		rf := rv.Field(i) // Value of the struct field (for reading actual value)

		if strLen, ok := sf.Tag.Lookup("len"); ok {
			numBitsToWrite, _ := strconv.Atoi(strLen)
			switch rf.Kind() {
			case reflect.Int:
				err = writer.WriteBits(uint64(rf.Int()), numBitsToWrite)
			case reflect.Uint:
				err = writer.WriteBits(rf.Uint(), numBitsToWrite)
			case reflect.Bool:
				err = writer.WriteBit(bitstream.Bit(rf.Bool()))
			case reflect.Array:
				size := rf.Len()

				for i := 0; i < size; i++ {
					rfi := rf.Index(i)
					err = writer.WriteBits(rfi.Uint(), numBitsToWrite)
					if err != nil {
						break
					}
				}
			case reflect.Struct:
				err = writeBitstreamFromReflection(writer, rf.Type(), rf)
				if err != nil {
					break
				}
			default:
				err = errors.New(fmt.Sprintf("Unhandled type discovered: %v\n", rf.Kind()))
			}
		} else {
			err = errors.New(fmt.Sprintf("Length for %s not specified, not sure how to proceed!", sf.Name))
		}

		if err != nil {
			break
		}
	}
	return err
}

// Consumes <length> unaligned bits from the bitstream and populates the reflect.Value as a Uint (of any size)
// Returns an error if one occurred
func readUint(into reflect.Value, from *bitstream.BitReader, length int) error {
	bits, err := from.ReadBits(length)
	if err != nil {
		return err
	}
	into.SetUint(uint64(bits))

	return nil
}

func readBool(into reflect.Value, from *bitstream.BitReader) error {
	bits, err := from.ReadBits(1)
	if err != nil {
		return err
	}
	into.SetBool(bits == 1)

	return nil
}

func readInt(into reflect.Value, from *bitstream.BitReader, length int) error {
	bits, err := from.ReadBits(1)
	if err != nil {
		return err
	}
	into.SetInt(int64(bits))

	return nil
}

func readUnaligned(from *bitstream.BitReader, length int) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	writer := bitstream.NewWriter(buf)

	// Currently we only support lengths in even bytes,
	// but we still read them unaligned (bitwise) from the reader.
	for i := 0; i < length/8; i++ {
		byteRead, err := from.ReadByte()
		if err != nil {
			return buf.Bytes(), err
		}
		writer.WriteByte(byteRead)
	}
	return buf.Bytes(), nil
}

func checksum8(payload []byte) byte {
	var runningSum uint8

	for _, currByte := range payload {
		runningSum += uint8(currByte)
	}

	return runningSum
}
