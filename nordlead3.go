package nordlead3

/*
TODO:

 - Expand slot concept to nl3edit too (move, rename, delete)
 - Try to identify the difference between v1.18 and v1.20 Sysex and see if you can figure out where the missing arp sync settings are.
 - Add two sane init patches (an init and an initFM), both with the right sync bits set.
*/

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/dgryski/go-bitstream"
)

const (
	strUninitializedName = "** Uninitialized"
)

// Position is also the sysex category value as a uint16 (0x00 to 0x0E observed)
var Categories = [15]string{
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
	"User2",
	"User3",
}

var (
	ErrXferTypeMismatch   = errors.New("Cannot move different types of patches")
	ErrInvalidLocation    = errors.New("Invalid location")
	ErrUninitialized      = errors.New("That location is not initialized")
	ErrInvalidCategory    = errors.New("Invalid category")
	ErrInvalidName        = errors.New("Name cannot be blank nor exceed 16 characters")
	ErrMemoryOccupied     = errors.New("One or more destination memory locations are not blank")
	ErrMemoryOverflow     = errors.New("Not enough room in that bank")
	ErrNoDataToWrite      = errors.New("No data to write to file")
	ErrNoPerfCategory     = errors.New("Performances do not support categories.")
	ErrImportTypeMismatch = errors.New("Sysex does not contain the right kind of patch (e.g. program when expecting performance).")
)

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
	var writer strings.Builder
	fprintStruct(&writer, s, depth)
	fmt.Print(writer.String())
}

func fprintStruct(writer io.Writer, s interface{}, depth int) {
	rv := reflect.ValueOf(s).Elem()
	rt := rv.Type()

	fprintReflectedStruct(writer, rt, rv, 0, depth)
}

func fprintReflectedStruct(writer io.Writer, rt reflect.Type, rv reflect.Value, indent int, depth int) {
	nameWidth, typeWidth := maxFieldWidths(rt, rv)

	for i := 0; i < rv.NumField(); i++ {
		sf := rt.Field(i)
		rf := rv.Field(i)

		fprintReflectedField(writer, sf, rf, indent, depth, nameWidth, typeWidth)
	}
}

func fprintReflectedField(writer io.Writer, sf reflect.StructField, rf reflect.Value, indent int, depth int, nameWidth int, typeWidth int) {
	strIndent := strings.Repeat(" ", indent*2)

	fmt.Fprintf(writer, "  %s%-*s (%*s): ", strIndent, nameWidth, sf.Name, typeWidth, sf.Type)

	switch rf.Kind() {
	case reflect.Int:
		fmt.Fprintf(writer, "%#02x / %d", rf.Int(), rf.Int())
	case reflect.Uint:
		fmt.Fprintf(writer, "%#02x / %d", rf.Uint(), rf.Uint())
	case reflect.Bool:
		fmt.Fprintf(writer, "%t", rf.Bool())
	case reflect.Array:
		fprintArrayToString(writer, rf)
	case reflect.Struct:
		fmt.Fprint(writer, " {")
		newStruct := reflect.New(sf.Type)
		if depth == 0 {
			fmt.Fprint(writer, " <hidden: beyond depth> }")
		} else {
			fmt.Fprintln(writer, "")
			fprintReflectedStruct(writer, newStruct.Elem().Type(), rf, indent+1, depth-1)
			fmt.Fprintf(writer, "  %s}", strIndent)
		}
	default:
		fmt.Fprintf(writer, "** Unhandled type discovered: %v", rf.Kind())
	}
	fmt.Fprint(writer, "\n")
}

func maxFieldWidths(rt reflect.Type, rv reflect.Value) (nameWidth int, typeWidth int) {
	mw := 0
	tw := 0

	for i := 0; i < rt.NumField(); i++ {
		sf := rt.Field(i)
		rf := rv.Field(i)

		// Structs have long names and we don't align them so we ignore them.
		if rf.Kind() != reflect.Struct {
			mw = max(mw, len(sf.Name))
			typeName := fmt.Sprintf("%s", sf.Type)
			typeWidth := len(typeName)
			tw = max(tw, typeWidth)
		}
	}

	return mw, tw
}

func fprintArrayToString(writer io.Writer, rv reflect.Value) {
	fmt.Fprint(writer, "[")
	size := rv.Len()
	strData := make([]string, 0)
	charData := make([]string, 0)

	for i := 0; i < size; i++ {
		rvi := rv.Index(i)
		strData = append(strData, fmt.Sprintf("%02x", rvi.Uint()))
		charData = append(charData, fmt.Sprintf("%q", rvi.Uint()))
	}
	fmt.Fprintf(writer, "%s] : [", strings.Join(strData, " "))
	fmt.Fprintf(writer, "%s]\n", strings.Join(charData, " "))
}
