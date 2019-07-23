package nordlead3

/*
TODO:

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
	// "math"
	"reflect"
	"strconv"

	"github.com/dgryski/go-bitstream"
)

func populateStructFromBitstream(i interface{}, data []byte) error {
	// Use reflection to get each field in the struct and it's length, then read that into it

	rt := reflect.TypeOf(i).Elem()
	rv := reflect.ValueOf(i).Elem()

	return populateReflectedStructFromBitstream(rt, rv, data)
}

func populateReflectedStructFromBitstream(rt reflect.Type, rv reflect.Value, data []byte) error {
	reader := bitstream.NewReader(bytes.NewReader(data))
	err := (error)(nil)
	// bitIx := 0
	// byteIx := 0

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i) // Type of the StructField (for reading tags)
		rf := rv.Field(i) // Value of the struct field (for setting value)

		if strLen, ok := sf.Tag.Lookup("len"); ok {
			numBitsToRead, _ := strconv.Atoi(strLen)
			switch rf.Kind() {
			case reflect.Int:
				err = readInt(rf, reader, numBitsToRead)
				// fmt.Printf("%4d:%-4d Read %d bits into %s (%s): %x\n", bitIx, byteIx, numBitsToRead, sf.Name, sf.Type, rf.Int())
			case reflect.Uint:
				err = readUint(rf, reader, numBitsToRead)
				// fmt.Printf("%4d:%-4d Read %d bits into %s (%s): %x\n", bitIx, byteIx, numBitsToRead, sf.Name, sf.Type, rf.Uint())
			case reflect.Bool:
				err = readBool(rf, reader)
				// fmt.Printf("%4d:%-4d Read %d bits into %s (%s): %t\n", bitIx, byteIx, numBitsToRead, sf.Name, sf.Type, rf.Bool())
			case reflect.Array:
				size := rf.Len()

				for i := 0; i < size; i++ {
					rfi := rf.Index(i)
					err = readUint(rfi, reader, numBitsToRead)
					// fmt.Printf("%4d:%-4d > Read %d bits into %s (%s): %x\n", bitIx, byteIx, numBitsToRead, sf.Name, sf.Type, rfi.Uint())
					if err != nil {
						break
					}
				}
			case reflect.Struct:
				// fmt.Println(" >> Diving into ", sf.Name)
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
			// bitIx += numBitsToRead
			// byteIx = int(math.Ceil(float64(bitIx) / 8.0))
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

	// err := writeBitstreamFromReflection(writer, rt, rv, buf)
	err := writeBitstreamFromReflection(writer, rt, rv)
	writer.Flush(bitstream.Zero)
	return buf.Bytes(), err
}

// func writeBitstreamFromReflection(writer *bitstream.BitWriter, rt reflect.Type, rv reflect.Value, buf *bytes.Buffer) error {
func writeBitstreamFromReflection(writer *bitstream.BitWriter, rt reflect.Type, rv reflect.Value) error {
	err := (error)(nil)
	// bitIx := 0
	// byteIx := 0

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i) // Type of the StructField (for reading tags)
		rf := rv.Field(i) // Value of the struct field (for reading actual value)

		if strLen, ok := sf.Tag.Lookup("len"); ok {
			numBitsToWrite, _ := strconv.Atoi(strLen)
			switch rf.Kind() {
			case reflect.Int:
				err = writer.WriteBits(uint64(rf.Int()), numBitsToWrite)
				// p1, _ := writer.Pending()
				// fmt.Printf("%4d:%-4d Wrote %2d bits from %21s (%4s): %#02x | %x | %02x\n", bitIx, byteIx, numBitsToWrite, sf.Name, sf.Type, uint8(rf.Int()), tail(buf, 16), p1)
			case reflect.Uint:
				err = writer.WriteBits(rf.Uint(), numBitsToWrite)
				// p1, _ := writer.Pending()
				// fmt.Printf("%4d:%-4d Wrote %2d bits from %21s (%4s): %#02x | %x | %02x\n", bitIx, byteIx, numBitsToWrite, sf.Name, sf.Type, rf.Uint(), tail(buf, 16), p1)
			case reflect.Bool:
				err = writer.WriteBit(bitstream.Bit(rf.Bool()))
				// p1, _ := writer.Pending()
				// fmt.Printf("%4d:%-4d Wrote %2d bits from %21s (%4s): %#02x | %x | %02x\n", bitIx, byteIx, numBitsToWrite, sf.Name, sf.Type, func() int {
				// 	if rf.Bool() {
				// 		return 1
				// 	} else {
				// 		return 0
				// 	}
				// }(), tail(buf, 16), p1)
			case reflect.Array:
				size := rf.Len()

				for i := 0; i < size; i++ {
					rfi := rf.Index(i)
					err = writer.WriteBits(rfi.Uint(), numBitsToWrite)
					// p1, _ := writer.Pending()
					// fmt.Printf("%4d:%-4d > Wrote %2d bits from %21s (%4s): %#02x | %x | %02x\n", bitIx, byteIx, numBitsToWrite, sf.Name, sf.Type, rfi.Uint(), tail(buf, 16), p1)
					if err != nil {
						break
					}
				}
			case reflect.Struct:
				err = writeBitstreamFromReflection(writer, rf.Type(), rf)
				// fmt.Println(" >> Diving into ", sf.Name)
				// err = writeBitstreamFromReflection(writer, rf.Type(), rf, buf)

				if err != nil {
					break
				}
			default:
				err = errors.New(fmt.Sprintf("Unhandled type discovered: %v\n", rf.Kind()))
			}
			// bitIx += numBitsToWrite
			// byteIx = int(math.Ceil(float64(bitIx) / 8.0))
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
	bits, err := from.ReadBits(length)
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
	numBytesToRead := length / 8
	if length%8 > 0 {
		panic("Reading lengths not evenly divisible by 8 is not yet supported.")
	}
	for i := 0; i < numBytesToRead; i++ {
		byteRead, err := from.ReadByte()
		if err != nil {
			return buf.Bytes(), err
		}
		writer.WriteByte(byteRead)
	}
	// fmt.Printf(" >> Read %d bytes: %x\n", numBytesToRead, buf.Bytes())
	return buf.Bytes(), nil
}

func checksum8(payload []byte) byte {
	var runningSum uint8

	for _, currByte := range payload {
		runningSum += uint8(currByte)
	}

	return runningSum
}

// func tail(buf *bytes.Buffer, n int) []byte {
// 	b := buf.Bytes()
// 	start := max(0, len(b)-n)
// 	return b[start:]
// }
