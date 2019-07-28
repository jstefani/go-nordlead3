package nordlead3

/*
TODO:

 - Get rid of the slotProgramT and slotPerformanceT types.
     - Instead, add a location in the ref that's either a slot or memory
     - The ref responds to source with either slotT or memoryT
     - Either type responds to index()
     - Because bank/location aren't shared by all types of ref
     - Interface Reference{} requires source() sourceType, index() int, contents() patchType
     - Helper methods RefFromSlot(patchType, index) and RefFromMemory(patchType, bank, location) can be public.
     - memory.get(Reference) should return a *patch, but should not be exported (patches aren't useful to consumers)
     - methods accepting indices should not be public, as many methods as possible, both internal and external, should use refs, if this reduces the number of methods needed
 - Find an interface abstraction that lets you do away with the custom behaviours for program/performance as much as possible
     - memory.get needs to be able to return one of these, perhaps "patch" or "patchable"?
     - Should not be exported, it doesn't have a real use outside the library functions.
     - Exported stuff should definitely be Program/Performance split
 - Be able to print and dump the slot content
 - Expand slot concept to nl3edit too (move, rename, delete)
 - Write a bunch of useful tests for the core methods
     - Test the move to/from slot methods
     - Test the delete methods
 - Re-implement move as a copy and a delete, creating copy method as well.
 - Create useful functions for manipulating memory:
     - Swap locations (separate from copy/move)
     - Insert a location (move following locations down until an empty location is hit, or return an error if there's no room)
 - Try to identify the difference between v1.18 and v1.20 Sysex and see if you can figure out where the missing arp sync settings are.
*/

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/dgryski/go-bitstream"
)

const (
	strUninitializedName = "** Uninitialized"
)

// Position is also the sysex category value as a uint16 (0x00 to 0x0D)
var Categories = [14]string{
	"Acoustic", // 0x00
	"Arpeggio", // 0x01
	"Bass",     // ...
	"Classic",
	"Drum",
	"Fantasy",
	"FX",
	"Lead",
	"Organ",
	"Pad",
	"Piano",
	"Synth",
	"User1",
	"User2", // 0x0D
}

var (
	ErrXferTypeMismatch = errors.New("Cannot move different types of patches")
	ErrInvalidLocation  = errors.New("Invalid location")
	ErrUninitialized    = errors.New("That location is not initialized")
	ErrInvalidCategory  = errors.New("Invalid category")
	ErrInvalidName      = errors.New("Name cannot be blank nor exceed 16 characters")
	ErrMemoryOccupied   = errors.New("One or more destination memory locations are not blank")
	ErrMemoryOverflow   = errors.New("Not enough room in that bank")
	ErrNoDataToWrite    = errors.New("No data to write to file")
)

func exportToFile(data *[]byte, filename string, overwrite bool) error {
	_, err := os.Stat(filename)
	if !os.IsNotExist(err) {
		if err != nil {
			return err
		}
		if !overwrite {
			return os.ErrExist
		}
	}

	file, err := os.Create(filename)
	fmt.Printf("Preparing %q\n", filename)
	if err != nil {
		return err
	}

	_, err = file.Write(*data)
	return err
}

func populateStructFromBitstream(i interface{}, data []byte) error {
	// Use reflection to get each field in the struct and it's length, then read that into it
	rt := reflect.TypeOf(i).Elem()
	rv := reflect.ValueOf(i).Elem()

	return populateReflectedStructFromBitstream(rt, rv, data, 0)
}

func populateReflectedStructFromBitstream(rt reflect.Type, rv reflect.Value, data []byte, depth int) error {
	// fmt.Printf("Populating %s with %x\n", rt.Name(), data)

	reader := bitstream.NewReader(bytes.NewReader(data))
	err := (error)(nil)

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i) // Type of the StructField (for reading tags)
		rf := rv.Field(i) // Value of the struct field (for setting value)

		if skipField(sf, depth) {
			continue
		}

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
				err = readArray(rf, reader, numBitsToRead)
			case reflect.Struct:
				err = readStruct(rf, sf, reader, numBitsToRead, depth)
			default:
				return errors.New(fmt.Sprintf("Unhandled type discovered: %v\n", rf.Kind()))
			}
		} else {
			err = errors.New(fmt.Sprintf("Length for %s not specified, not sure how to proceed!", sf.Name))
		}

		if err == io.EOF {
			err = errors.New(fmt.Sprintf("EOF parsing field %q in %q.", sf.Name, rt.Name()))
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

	err := writeBitstreamFromReflection(writer, rt, rv, 0)
	writer.Flush(bitstream.Zero)
	return buf.Bytes(), err
}

func writeBitstreamFromReflection(writer *bitstream.BitWriter, rt reflect.Type, rv reflect.Value, depth int) error {
	err := (error)(nil)

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i) // Type of the StructField (for reading tags)
		rf := rv.Field(i) // Value of the struct field (for reading actual value)

		if skipField(sf, depth) {
			continue
		}

		if strLen, ok := sf.Tag.Lookup("len"); ok {
			numBitsToWrite, _ := strconv.Atoi(strLen)
			err = writeReflectedType(writer, rf, numBitsToWrite, depth)
		} else {
			err = errors.New(fmt.Sprintf("Length for %s not specified, not sure how to proceed!", sf.Name))
		}

		if err != nil {
			break
		}
	}

	return err
}

func writeReflectedType(writer *bitstream.BitWriter, rf reflect.Value, numBitsToWrite int, depth int) error {
	err := (error)(nil)

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
		err = writeBitstreamFromReflection(writer, rf.Type(), rf, depth+1)
	default:
		err = errors.New(fmt.Sprintf("Unhandled type discovered: %v\n", rf.Kind()))
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

func readArray(into reflect.Value, from *bitstream.BitReader, length int) error {
	size := into.Len()

	for i := 0; i < size; i++ {
		elem := into.Index(i)
		err := readUint(elem, from, length)
		if err != nil {
			return err
		}
	}

	return nil
}

func readStruct(into reflect.Value, field reflect.StructField, from *bitstream.BitReader, length int, depth int) error {
	bitstream, err := readUnaligned(from, length)
	if err == nil {
		newStruct := reflect.New(field.Type)
		err = populateReflectedStructFromBitstream(newStruct.Elem().Type(), newStruct.Elem(), bitstream, depth+1)
		if err == nil {
			into.Set(newStruct.Elem())
		}
	}
	return err
}

func skipField(field reflect.StructField, depth int) bool {
	if _, ok := field.Tag.Lookup("skipEmbedded"); ok {
		if depth > 0 {
			return true
		}
	}
	return false
}

func readUnaligned(from *bitstream.BitReader, length int) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	writer := bitstream.NewWriter(buf)

	numBytesToRead := length / 8
	numTrailingBitsToRead := length % 8

	for i := 0; i < numBytesToRead; i++ {
		byteRead, err := from.ReadByte()
		if err != nil {
			return buf.Bytes(), err
		}
		writer.WriteByte(byteRead)
	}
	for i := 0; i < numTrailingBitsToRead; i++ {
		bit, err := from.ReadBit()
		if err != nil {
			return buf.Bytes(), err
		}
		writer.WriteBit(bit)
	}
	writer.Flush(bitstream.Zero)
	// fmt.Printf(" >> Read %d bytes and %d bits (%d bits in total): %x\n", numBytesToRead, numTrailingBitsToRead, length, buf.Bytes())
	return buf.Bytes(), nil
}

func checksum8(payload []byte) byte {
	var runningSum uint8

	for _, currByte := range payload {
		runningSum += uint8(currByte)
	}

	return runningSum
}

func printStruct(s interface{}, depth int) {
	rv := reflect.ValueOf(s).Elem()
	rt := rv.Type()

	printReflectedStruct(rt, rv, 0, depth)
}

func printReflectedStruct(rt reflect.Type, rv reflect.Value, indent int, depth int) {
	nameWidth, typeWidth := maxFieldWidths(rt, rv)

	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		rf := rv.Field(i)

		printReflectedField(sf, rf, indent, depth, nameWidth, typeWidth)
	}
}

func printReflectedField(sf reflect.StructField, rf reflect.Value, indent int, depth int, nameWidth int, typeWidth int) {
	strIndent := strings.Repeat(" ", indent*2)

	fmt.Printf("  %s%-*s (%*s): ", strIndent, nameWidth, sf.Name, typeWidth, sf.Type)

	switch rf.Kind() {
	case reflect.Int:
		fmt.Printf("%#02x / %d", rf.Int(), rf.Int())
	case reflect.Uint:
		fmt.Printf("%#02x / %d", rf.Uint(), rf.Uint())
	case reflect.Bool:
		fmt.Printf("%t", rf.Bool())
	case reflect.Array:
		printArrayToString(rf)
	case reflect.Struct:
		fmt.Print(" {")
		newStruct := reflect.New(sf.Type)
		if depth == 0 {
			fmt.Print(" <hidden: beyond depth> }")
		} else {
			fmt.Println("")
			printReflectedStruct(newStruct.Elem().Type(), rf, indent+1, depth-1)
			fmt.Printf("  %s}", strIndent)
		}
	default:
		fmt.Sprintf("** Unhandled type discovered: %v", rf.Kind())
	}
	fmt.Print("\n")
}

func maxFieldWidths(rt reflect.Type, rv reflect.Value) (nameWidth int, typeWidth int) {
	mw := 0
	tw := 0

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		rf := rv.Field(i)

		// Structs have long names and we don't align them so we ignore them.
		if rf.Kind() != reflect.Struct {
			mw = Max(mw, len(sf.Name))
			typeName := fmt.Sprintf("%s", sf.Type)
			typeWidth := len(typeName)
			tw = Max(tw, typeWidth)
		}
	}

	return mw, tw
}

func printArrayToString(rv reflect.Value) {
	fmt.Print("[")
	size := rv.Len()
	strData := make([]string, 0)
	charData := make([]string, 0)

	for i := 0; i < size; i++ {
		rvi := rv.Index(i)
		strData = append(strData, fmt.Sprintf("%02x", rvi.Uint()))
		charData = append(charData, fmt.Sprintf("%q", rvi.Uint()))
	}
	fmt.Printf("%s] : [", strings.Join(strData, " "))
	fmt.Printf("%s]\n", strings.Join(charData, " "))
}
