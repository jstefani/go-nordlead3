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
		// TODO actually parse the sysex into the right object type and stick in the memory array
		fmt.Printf("Loaded %s: (%v:%03d) %-16.16q v%1.2f\n", sysex.printableType(), sysex.bank(), sysex.location(), sysex.printableName(), sysex.version())
	}

	return err
}

type ProgramBank struct {
	id       uint
	programs [128]Program
}

type PerformanceBank struct {
	id           uint
	performances [128]Performance
}

type Program struct {
	version_number        uint16      // Decimal OS version number (# x	100	)
	osc1_shape            uint8       // 0-127
	osc2_coarse_pitch     uint8       // 0-127
	osc2_fine_pitch       uint8       // 0-127
	osc2_shape            uint8       // 0-127
	oscmix                uint8       // 0-127
	oscmod                uint8       // 0-127
	lfo1_rate             uint8       // 0-127
	lfo1_amount           uint8       // 0-127
	lfo2_rate             uint8       // 0-127
	lfo2_amount           uint8       // 0-127
	amp_env_attack        uint8       // 0-127
	amp_env_decay         uint8       // 0-127
	amp_env_sustain       uint8       // 0-127
	amp_env_release       uint8       // 0-127
	output_level          uint8       // 0-127
	filt_env_attack       uint8       // 0-127
	filt_env_decay        uint8       // 0-127
	filt_env_sustain      uint8       // 0-127
	filt_env_release      uint8       // 0-127
	mod_env_attack        uint8       // 0-127
	mod_env_decay_release uint8       // 0-127
	mod_env_amount        uint8       // 0-127
	filt_env_amount       uint8       // 0-127
	filt_frequency1       uint8       // 0-127
	filt_resonance        uint8       // 0-127
	filt_frequency2       uint8       // 0-127
	unison_amount         uint8       // 0-127
	filt_dist_amount      uint8       // 0-127
	osc1_sync_tune        uint8       // 0-127
	osc2_sync_tune        uint8       // 0-127
	osc1_noise_seed       uint8       // 0-127
	osc2_noise_seed       uint8       // 0-127
	osc1_modulator_amount uint8       // 0-127 In Dual Sine mode
	osc2_modulator_amount uint8       // 0-127 In Dual Sine mode
	osc2_carrier_pitch    uint8       // 0-127 In Dual Sine mode
	osc2_noise_type       uint8       // 0-127 LP/BP/HP
	osc2_modulator_pitch  uint8       // 0-127 In Dual Sine mode
	osc2_noise_frequency  uint8       // 0-127
	spare1                uint8       // 0-255
	spare2                uint8       // 0-255
	glide_rate            uint8       // 0-127
	arpeggio_rate         uint8       // 0-127
	vibrato_rate          uint8       // 0-127
	vibrato_amount        uint8       // 0-127
	arpeggio_sync_divisor uint8       // 0-127
	lfo1_sync_divisor     uint8       // 0-127
	lfo2_sync_divisor     uint8       // 0-127
	transpose             uint8       // 0-127
	spare3                uint8       // 0-255
	spare4                uint8       // 0-255
	osc1_waveform         uint8       // 0-5
	osc1_sync             bool        // 0-1
	osc2_waveform         uint8       // 0-5
	osc2_sync             bool        // 0-1
	osc2_kbt              bool        // 0-1 Off or On
	osc2_partial          bool        // 0-1 Off or On
	oscmod_type           uint8       // 0-5
	lfo1_waveform         uint8       // 0-5
	lfo1_destination      uint8       // 0-11
	lfo1_env_kbs          uint8       // 0-2
	lfo1_mono             bool        // 0-1 Off or On
	lfo1_invert           bool        // 0-1 Off or On
	lfo2_waveform         uint8       // 0-5
	lfo2_destination      uint8       // 0-11
	lfo2_env_kbs          uint8       // 0-2
	lfo2_mono             bool        // 0-1 Off or On
	lfo2_invert           bool        // 0-1 Off or On
	mod_env_invert        bool        // 0-1 Off or On
	mod_env_destination   uint8       // 0-11
	mod_env_mode          bool        // 0-1 Selects Decay (0) or Release (1) mode
	mod_env_repeat        bool        // 0-1 Off or On
	filt1_type            uint8       // 0-5
	filt1_slope           uint8       // 0-2
	filt_env_velocity     bool        // 0-1 Off or On
	filt1_kbt             bool        // 0-1 Off or On
	filt_env_invert       bool        // 0-1 Off or On
	amp_env_exp_attack    bool        // 0-1 Off or On
	mod_env_exp_attack    bool        // 0-1 Off or On
	filt_env_exp_attack   bool        // 0-1 Off or On
	filt_mode             bool        // 0-1 Selects Single or Dual Filter mode
	filt2_env             bool        // 0-1 Selects if Filt2 is controlled by Filt_Env
	filt2_type            uint8       // 0-5
	filt_bypass           bool        // 0-1 Off or On
	lfo1_clocksync        bool        // 0-1 Off or On
	lfo2_clocksync        bool        // 0-1 Off or On
	arpeggiator_clocksync bool        // 0-1 Off or On
	oscmix_noise          bool        // 0-1 Off or On
	glide_mode            uint8       // 0-2 Off, On or Auto
	vibrato_source        uint8       // 0-2 Off, Wheel or Aftertouch
	mono_mode             bool        // 0-1 Off or On
	arpeggio_run          bool        // 0-1 Off or On
	spare5                uint8       // 0-255
	unison_mode           bool        // 0-1 Off or On
	octave_shift          uint8       // 0-4
	chord_mem_mode        bool        // 0-1 Off or On
	arpeggio_mode         uint8       // 0-3 Up, Down, Up/down or Random
	arpeggio_range        uint8       // 0-3
	arpeggio_kbd_sync     bool        // 0-1 Off or On
	spare6                uint8       // 0-255
	spare7                bool        // 0-1 Off or On
	legato_mode           bool        // 0-1 Off or On
	mono_allocation_mode  uint8       // 0-2 Off, Hi or Lo
	wheel_morph_params    MorphParams // 0-127 See ‘Morph parameter list’ below
	a_touch_morph_params  MorphParams // 0-127 See ‘Morph parameter list’ below
	velocity_morph_params MorphParams // 0-127 See ‘Morph parameter list’ below
	kbd_morph_params      MorphParams // 0-127 See ‘Morph parameter list’ below
	chord_mem_count       uint8       // 0-23	uint8	-24
	chord_mem_position    uint8       // 0-255	uint8	-24
	spare8                uint8       // 0-255
	checksum              uint8       // 0-255
}

type MorphParams struct {
	// Morph Parameter List (all parameters -128 to 127)
	lfo1_rate             int8
	lfo1_amount           int8
	lfo2_rate             int8
	lfo2_amount           int8
	mod_env_attack        int8
	mod_env_decay_release int8
	mod_env_amount        int8
	osc2_fine_pitch       int8
	osc2_coarse_pitch     int8
	oscmod                int8
	oscmix                int8
	osc1_shape            int8
	osc2_shape            int8
	amp_env_attack        int8
	amp_env_decay         int8
	amp_env_sustain       int8
	amp_env_release       int8
	filt_env_attack       int8
	filt_env_decay        int8
	filt_env_sustain      int8
	filt_env_release      int8
	filt_env_amount       int8
	filt_frequency1       int8
	filt_frequency2       int8
	filt_resonance        int8
	output_level          int8
}

type Performance struct {
	version_number       uint16    // Decimal OS version number (# x	100	)
	enabled_slots        uint8     // 0-15
	focused_slot         uint8     // 0-3
	midi_channel_slot_a  uint8     // 0-16 0 = Off
	midi_channel_slot_b  uint8     // 0-16 0 = Off
	midi_channel_slot_c  uint8     // 0-16 0 = Off
	midi_channel_slot_d  uint8     // 0-16 0 = Off
	audio_channel_slot_a uint8     // 0-5
	audio_channel_slot_b uint8     // 0-5
	audio_channel_slot_c uint8     // 0-5
	audio_channel_slot_d uint8     // 0-5
	splitpoint_key       uint8     // 0-127
	splitpoint           uint8     // 0-1 Off or On
	sustain_enable       uint8     // 0-15
	pitchbend_enable     uint8     // 0-15
	modwheel_enable      uint8     // 0-15
	bank_slot_a          uint8     // 0-7
	program_slot_a       uint8     // 0-127
	bank_slot_b          uint8     // 0-7
	program_slot_b       uint8     // 0-127
	bank_slot_c          uint8     // 0-7
	program_slot_c       uint8     // 0-127
	bank_slot_d          uint8     // 0-7
	program_slot_d       uint8     // 0-127
	morph3_source_select uint8     // 0-1 Control pedal or aftertouch
	midi_clock_keysync   uint8     // 0-1 Off or On
	keyboard_hold        uint8     // 0-1 Off or On
	spare3               uint8     // Always set to zero
	spare4               uint8     // Always set to zero
	spare5               uint8     // Always set to zero
	spare6               uint8     // Always set to zero
	spare7               uint8     // Always set to zero
	spare8               uint8     // Always set to zero
	spare9               uint8     // Always set to zero
	spare10              uint8     // Always set to zero
	spare11              uint8     // Always set to zero
	spare12              uint8     // Always set to zero
	midi_clock_rate      uint16    // 0-210
	bend_range_up        uint8     // 0-24
	bend_range_down      uint8     // 0-24
	patchname_slot_a     [16]uint8 // Max 16 characters long
	patchname_slot_b     [16]uint8 // Max 16 characters long
	patchname_slot_c     [16]uint8 // Max 16 characters long
	patchname_slot_d     [16]uint8 // Max 16 characters long
	patch_data           [4]Program
	checksum             uint8
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
