package nordlead3

/*
TODO:

 - Handle reading the byte arrays for the patch names in performances
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
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/dgryski/go-bitstream"
)

const (
	ProgramFromSlot       = 0x20
	ProgramFromMemory     = 0x21
	PerformanceFromSlot   = 0x28
	PerformanceFromMemory = 0x29
)

const (
	VersionOffset   = 38
	PatchDataOffset = 40
)

const (
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
	// MIDI 8-bit to bitstream decoding
	// Every byte of the MIDI stream is actually only 7 bits of the payload bitstream
	// so we need to drop a bit every byte and re-concatenate the bits
	payload := sysex.rawSysex[PatchDataOffset:]
	buf := bytes.NewBuffer(nil)
	reader := bitstream.NewReader(strings.NewReader(string(payload)))
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

	sysex.decodedBitstream = buf.Bytes()
}

func (sysex *Sysex) location() uint8 {
	return sysex.rawSysex[5]
}

func (sysex *Sysex) messageType() uint8 {
	return sysex.rawSysex[3]
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
	var runningSum uint8
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
	for _, currByte := range payload {
		runningSum += uint8(currByte)
	}
	if checksum != runningSum {
		errStrs = append(errStrs, fmt.Sprintf("Checksum mismatch parsing %s (%v:%03d %q): expected %x, got %x", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), checksum, runningSum))
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

// PatchMemory holds the entire internal structure of the patch memory, including locations, names, and patch contents.
// The main object responsible for organizing programs and performances.

type PatchMemory struct {
	programs     [8]ProgramBank
	performances [2]PerformanceBank
}

func (memory *PatchMemory) LoadFromSysex(sysex *Sysex) error {
	err := *new(error)

	valid, err := sysex.valid()

	if valid {
		switch sysex.messageType() {
		case ProgramFromMemory, ProgramFromSlot:
			memory.LoadProgramFromSysex(sysex)
		case PerformanceFromMemory, PerformanceFromSlot:
			memory.LoadPerformanceFromSysex(sysex)
		}
	}

	return err
}

func (memory *PatchMemory) LoadPerformanceFromSysex(sysex *Sysex) {
	performance, err := NewPerformanceFromBitstream(sysex.decodedBitstream)
	if err == nil {
		perfLocation := PerformanceLocation{Name: sysex.nameAsArray(), Version: sysex.version(), Performance: performance}
		memory.performances[sysex.bank()].performances[sysex.location()] = &perfLocation
		fmt.Printf("Loaded %s: (%v:%03d) %-16.16q v%1.2f\n", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), sysex.version())
	} else {
		panic(err)
	}
}

func (memory *PatchMemory) LoadProgramFromSysex(sysex *Sysex) {
	program, err := NewProgramFromBitstream(sysex.decodedBitstream)
	if err == nil {
		programLocation := ProgramLocation{Name: sysex.nameAsArray(), Version: sysex.version(), Program: program}
		memory.programs[sysex.bank()].programs[sysex.location()] = &programLocation
		fmt.Printf("Loaded %s: (%v:%03d) %-16.16q v%1.2f\n", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), sysex.version())
	} else {
		panic(err)
	}
}

func (memory *PatchMemory) PrintPrograms(omitBlank bool) string {
	var result []string

	result = append(result, "\n***** PROGRAMS ******\n")
	for bank, contents := range memory.programs {
		bank_header := fmt.Sprintf("\n*** Bank %v ***\n", bank+1)
		result = append(result, bank_header)
		result = append(result, contents.PrintSummary(omitBlank))
	}

	return strings.Join(result, "\n")
}

func (memory *PatchMemory) PrintPerformances(omitBlank bool) string {
	var result []string

	result = append(result, "\n***** PERFORMANCES ******\n")

	for bank, contents := range memory.performances {
		bank_header := fmt.Sprintf("\n*** Bank %v ***\n", bank+1)
		result = append(result, bank_header)
		result = append(result, contents.PrintSummary(omitBlank))
	}

	return strings.Join(result, "\n")
}

type ProgramBank struct {
	id       uint
	programs [128]*ProgramLocation
}

func (bank *ProgramBank) PrintSummary(omitBlank bool) string {
	var result []string

	for location, program := range bank.programs {
		if program != nil {
			result = append(result, fmt.Sprintf("   %3d : %+-16.16q (%1.2f)", location, program.PrintableName(), program.Version))
		} else if !omitBlank {
			result = append(result, fmt.Sprintf("   %3d : %+-16.16q", location, program.PrintableName()))
		}
	}

	return strings.Join(result, "\n")
}

type PerformanceBank struct {
	id           uint
	performances [128]*PerformanceLocation
}

func (bank *PerformanceBank) PrintSummary(omitBlank bool) string {
	var result []string

	for location, performance := range bank.performances {
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
	Category uint
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
	Category    uint
	Version     float64
	Performance *Performance
}

func (perfLoc *PerformanceLocation) PrintableName() string {
	if perfLoc == nil {
		return "** Uninitialized"
	}
	return fmt.Sprintf("%-16s", strings.TrimRight(string(perfLoc.Name[:]), "\x00"))
}

type Program struct {
	Version_number        uint        `len:"16"`
	Osc1_shape            uint        `len:"7" min:"0" max:"127"`
	Osc2_coarse_pitch     uint        `len:"7" min:"0" max:"127"`
	Osc2_fine_pitch       uint        `len:"7" min:"0" max:"127"`
	Osc2_shape            uint        `len:"7" min:"0" max:"127"`
	Oscmix                uint        `len:"7" min:"0" max:"127"`
	Oscmod                uint        `len:"7" min:"0" max:"127"`
	Lfo1_rate             uint        `len:"7" min:"0" max:"127"`
	Lfo1_amount           uint        `len:"7" min:"0" max:"127"`
	Lfo2_rate             uint        `len:"7" min:"0" max:"127"`
	Lfo2_amount           uint        `len:"7" min:"0" max:"127"`
	Amp_env_attack        uint        `len:"7" min:"0" max:"127"`
	Amp_env_decay         uint        `len:"7" min:"0" max:"127"`
	Amp_env_sustain       uint        `len:"7" min:"0" max:"127"`
	Amp_env_release       uint        `len:"7" min:"0" max:"127"`
	Output_level          uint        `len:"7" min:"0" max:"127"`
	Filt_env_attack       uint        `len:"7" min:"0" max:"127"`
	Filt_env_decay        uint        `len:"7" min:"0" max:"127"`
	Filt_env_sustain      uint        `len:"7" min:"0" max:"127"`
	Filt_env_release      uint        `len:"7" min:"0" max:"127"`
	Mod_env_attack        uint        `len:"7" min:"0" max:"127"`
	Mod_env_decay_release uint        `len:"7" min:"0" max:"127"`
	Mod_env_amount        uint        `len:"7" min:"0" max:"127"`
	Filt_env_amount       uint        `len:"7" min:"0" max:"127"`
	Filt_frequency1       uint        `len:"7" min:"0" max:"127"`
	Filt_resonance        uint        `len:"7" min:"0" max:"127"`
	Filt_frequency2       uint        `len:"7" min:"0" max:"127"`
	Unison_amount         uint        `len:"7" min:"0" max:"127"`
	Filt_dist_amount      uint        `len:"7" min:"0" max:"127"`
	Osc1_sync_tune        uint        `len:"7" min:"0" max:"127"`
	Osc2_sync_tune        uint        `len:"7" min:"0" max:"127"`
	Osc1_noise_seed       uint        `len:"7" min:"0" max:"127"`
	Osc2_noise_seed       uint        `len:"7" min:"0" max:"127"`
	Osc1_modulator_amount uint        `len:"7" min:"0" max:"127"`
	Osc2_modulator_amount uint        `len:"7" min:"0" max:"127"`
	Osc2_carrier_pitch    uint        `len:"7" min:"0" max:"127"`
	Osc2_noise_type       uint        `len:"7" min:"0" max:"127"`
	Osc2_modulator_pitch  uint        `len:"7" min:"0" max:"127"`
	Osc2_noise_frequency  uint        `len:"7" min:"0" max:"127"`
	Spare1                uint        `len:"8" min:"0" max:"255"`
	Spare2                uint        `len:"8" min:"0" max:"255"`
	Glide_rate            uint        `len:"7" min:"0" max:"127"`
	Arpeggio_rate         uint        `len:"7" min:"0" max:"127"`
	Vibrato_rate          uint        `len:"7" min:"0" max:"127"`
	Vibrato_amount        uint        `len:"7" min:"0" max:"127"`
	Arpeggio_sync_divisor uint        `len:"7" min:"0" max:"127"`
	Lfo1_sync_divisor     uint        `len:"7" min:"0" max:"127"`
	Lfo2_sync_divisor     uint        `len:"7" min:"0" max:"127"`
	Transpose             uint        `len:"7" min:"0" max:"127"`
	Spare3                uint        `len:"8" min:"0" max:"255"`
	Spare4                uint        `len:"8" min:"0" max:"255"`
	Osc1_waveform         uint        `len:"3" min:"0" max:"5"`
	Osc1_sync             bool        `len:"1" min:"0" max:"1"`
	Osc2_waveform         uint        `len:"3" min:"0" max:"5"`
	Osc2_sync             bool        `len:"1" min:"0" max:"1"`
	Osc2_kbt              bool        `len:"1" min:"0" max:"1 O"`
	Osc2_partial          bool        `len:"1" min:"0" max:"1 O"`
	Oscmod_type           uint        `len:"3" min:"0" max:"5"`
	Lfo1_waveform         uint        `len:"3" min:"0" max:"5"`
	Lfo1_destination      uint        `len:"4" min:"0" max:"11"`
	Lfo1_env_kbs          uint        `len:"2" min:"0" max:"2"`
	Lfo1_mono             bool        `len:"1"`
	Lfo1_invert           bool        `len:"1"`
	Lfo2_waveform         uint        `len:"3" min:"0" max:"5"`
	Lfo2_destination      uint        `len:"4" min:"0" max:"11"`
	Lfo2_env_kbs          uint        `len:"2" min:"0" max:"2"`
	Lfo2_mono             bool        `len:"1"`
	Lfo2_invert           bool        `len:"1"`
	Mod_env_invert        bool        `len:"1"`
	Mod_env_destination   uint        `len:"4" min:"0" max:"11"`
	Mod_env_mode          bool        `len:"1"`
	Mod_env_repeat        bool        `len:"1"`
	Filt1_type            uint        `len:"3" min:"0" max:"5"`
	Filt1_slope           uint        `len:"2" min:"0" max:"2"`
	Filt_env_velocity     bool        `len:"1"`
	Filt1_kbt             bool        `len:"1"`
	Filt_env_invert       bool        `len:"1"`
	Amp_env_exp_attack    bool        `len:"1"`
	Mod_env_exp_attack    bool        `len:"1"`
	Filt_env_exp_attack   bool        `len:"1"`
	Filt_mode             bool        `len:"1"`
	Filt2_env             bool        `len:"1"`
	Filt2_type            uint        `len:"3" min:"0" max:"5"`
	Filt_bypass           bool        `len:"1"`
	Lfo1_clocksync        bool        `len:"1"`
	Lfo2_clocksync        bool        `len:"1"`
	Arpeggiator_clocksync bool        `len:"1"`
	Oscmix_noise          bool        `len:"1"`
	Glide_mode            uint        `len:"2" min:"0" max:”2"`
	Vibrato_source        uint        `len:"2" min:"0" max:"2"`
	Mono_mode             bool        `len:"1"`
	Arpeggio_run          bool        `len:"1"`
	Spare5                uint        `len:"8" min:"0" max:"255"`
	Unison_mode           bool        `len:"1"`
	Octave_shift          uint        `len:"3" min:"0" max:"4"`
	Chord_mem_mode        bool        `len:"1"`
	Arpeggio_mode         uint        `len:"3" min:"0" max:”3"`
	Arpeggio_range        uint        `len:"3" min:"0" max:"3"`
	Arpeggio_kbd_sync     bool        `len:"1"`
	Spare6                uint        `len:"8" min:"0" max:"255"`
	Spare7                bool        `len:"1"`
	Legato_mode           bool        `len:"1"`
	Mono_allocation_mode  uint        `len:"2" min:"0" max:"2"`
	Wheel_morph_params    MorphParams `len:"208"`
	A_touch_morph_params  MorphParams `len:"208"`
	Velocity_morph_params MorphParams `len:"208"`
	Kbd_morph_params      MorphParams `len:"208"`
	Chord_mem_count       uint        `len:"5" min:"0" max:"23"`
	Chord_mem_position    uint        `len:"8" min:"0" max:"255"`
	Spare8                uint        `len:"8"`
	Checksum              uint        `len:"8" min:"0" max:"255"`
}

type MorphParams struct {
	Lfo1_rate             int `len:"8" min:"-128" max:"127"`
	Lfo1_amount           int `len:"8" min:"-128" max:"127"`
	Lfo2_rate             int `len:"8" min:"-128" max:"127"`
	Lfo2_amount           int `len:"8" min:"-128" max:"127"`
	Mod_env_attack        int `len:"8" min:"-128" max:"127"`
	Mod_env_decay_release int `len:"8" min:"-128" max:"127"`
	Mod_env_amount        int `len:"8" min:"-128" max:"127"`
	Osc2_fine_pitch       int `len:"8" min:"-128" max:"127"`
	Osc2_coarse_pitch     int `len:"8" min:"-128" max:"127"`
	Oscmod                int `len:"8" min:"-128" max:"127"`
	Oscmix                int `len:"8" min:"-128" max:"127"`
	Osc1_shape            int `len:"8" min:"-128" max:"127"`
	Osc2_shape            int `len:"8" min:"-128" max:"127"`
	Amp_env_attack        int `len:"8" min:"-128" max:"127"`
	Amp_env_decay         int `len:"8" min:"-128" max:"127"`
	Amp_env_sustain       int `len:"8" min:"-128" max:"127"`
	Amp_env_release       int `len:"8" min:"-128" max:"127"`
	Filt_env_attack       int `len:"8" min:"-128" max:"127"`
	Filt_env_decay        int `len:"8" min:"-128" max:"127"`
	Filt_env_sustain      int `len:"8" min:"-128" max:"127"`
	Filt_env_release      int `len:"8" min:"-128" max:"127"`
	Filt_env_amount       int `len:"8" min:"-128" max:"127"`
	Filt_frequency1       int `len:"8" min:"-128" max:"127"`
	Filt_frequency2       int `len:"8" min:"-128" max:"127"`
	Filt_resonance        int `len:"8" min:"-128" max:"127"`
	Output_level          int `len:"8" min:"-128" max:"127"`
}

type Performance struct {
	Version_number       uint     `len:"16"`                  // Decimal OS version number (# x	100	)
	Enabled_slots        uint     `len:"8" min:"0" max:"127"` // 0-15
	Focused_slot         uint     `len:"8" min:"0" max:"127"` // 0-3
	Midi_channel_slot_a  uint     `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Midi_channel_slot_b  uint     `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Midi_channel_slot_c  uint     `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Midi_channel_slot_d  uint     `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Audio_channel_slot_a uint     `len:"8" min:"0" max:"127"` // 0-5
	Audio_channel_slot_b uint     `len:"8" min:"0" max:"127"` // 0-5
	Audio_channel_slot_c uint     `len:"8" min:"0" max:"127"` // 0-5
	Audio_channel_slot_d uint     `len:"8" min:"0" max:"127"` // 0-5
	Splitpoint_key       uint     `len:"8" min:"0" max:"127"` // 0-127
	Splitpoint           bool     `len:"8" min:"0" max:"127"` // 0-1 Off or On
	Sustain_enable       uint     `len:"8" min:"0" max:"127"` // 0-15
	Pitchbend_enable     uint     `len:"8" min:"0" max:"127"` // 0-15
	Modwheel_enable      uint     `len:"8" min:"0" max:"127"` // 0-15
	Bank_slot_a          uint     `len:"3" min:"0" max:"7"`
	Program_slot_a       uint     `len:"8" min:"0" max:"127"`
	Bank_slot_b          uint     `len:"3" min:"0" max:"7"`
	Program_slot_b       uint     `len:"8" min:"0" max:"127"`
	Bank_slot_c          uint     `len:"3" min:"0" max:"7"`
	Program_slot_c       uint     `len:"8" min:"0" max:"127"`
	Bank_slot_d          uint     `len:"3" min:"0" max:"7"`
	Program_slot_d       uint     `len:"8" min:"0" max:"127"`
	Morph3_source_select bool     `len:"8"` // 0-1 Control pedal or aftertouch
	Midi_clock_keysync   bool     `len:"8"`
	Keyboard_hold        bool     `len:"8"`
	Spare3               uint     `len:"8"`
	Spare4               uint     `len:"8"`
	Spare5               uint     `len:"8"`
	Spare6               uint     `len:"8"`
	Spare7               uint     `len:"8"`
	Spare8               uint     `len:"8"`
	Spare9               uint     `len:"8"`
	Spare10              uint     `len:"8"`
	Spare11              uint     `len:"8"`
	Spare12              uint     `len:"8"`
	Midi_clock_rate      uint     `len:"8" min:"0" max:"210"` // 0-210
	Bend_range_up        uint     `len:"8" min:"0" max:"24"`  // 0-24
	Bend_range_down      uint     `len:"8" min:"0" max:"24"`  // 0-24
	Patchname_slot_a     [16]byte `len:"7"`                   // Read as 16 chars of 7 bits, so read 7 bits into each of 16 bytes
	Patchname_slot_b     [16]byte `len:"7"`
	Patchname_slot_c     [16]byte `len:"7"`
	Patchname_slot_d     [16]byte `len:"7"`
	Patch_data_a         Program  `len:"191"`
	Patch_data_b         Program  `len:"191"`
	Patch_data_c         Program  `len:"191"`
	Patch_data_d         Program  `len:"191"`
	Checksum             uint     `len:"8"`
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

func NewProgramFromBitstream(data []byte) (*Program, error) {
	program := new(Program)
	err := populateStructFromBitstream(program, data)
	return program, err
}

func NewPerformanceFromBitstream(data []byte) (*Performance, error) {
	performance := new(Performance)
	err := populateStructFromBitstream(performance, data)
	return performance, err
}

func populateStructFromBitstream(i interface{}, data []byte) error {
	// Use reflection to get each field in the struct and it's length, then read that into it

	rt := reflect.TypeOf(i).Elem()
	rv := reflect.ValueOf(i).Elem()

	return populateReflectedStructFromBitstream(rt, rv, data)
}

func populateReflectedStructFromBitstream(rt reflect.Type, rv reflect.Value, data []byte) error {
	reader := bitstream.NewReader(strings.NewReader(string(data)))
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
