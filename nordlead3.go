package nordlead3

import (
	"bytes"
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
	VersionOffset         = 38
	PatchDataOffset       = 40
)

type PatchMemory struct {
	programs 		[8]ProgramBank
	performances	[2]PerformanceBank
}

type ProgramBank struct {
	id				uint
	programs		[128]Program
}

type PerformanceBank struct {
	id				uint
	performances	[128]Performance
}

type Program struct {
	version_number			uint16         // Decimal OS version number (# x	100	)
	osc1_shape				uint8          // 0-127 
	osc2_coarse_pitch		uint8          // 0-127 
	osc2_fine_pitch			uint8          // 0-127 
	osc2_shape				uint8          // 0-127 
	oscmix					uint8          // 0-127 
	oscmod					uint8          // 0-127 
	lfo1_rate				uint8          // 0-127 
	lfo1_amount				uint8          // 0-127 
	lfo2_rate				uint8          // 0-127 
	lfo2_amount				uint8          // 0-127 
	amp_env_attack			uint8          // 0-127 
	amp_env_decay			uint8          // 0-127 
	amp_env_sustain			uint8          // 0-127 
	amp_env_release			uint8          // 0-127 
	output_level			uint8          // 0-127 
	filt_env_attack			uint8          // 0-127 
	filt_env_decay			uint8          // 0-127 
	filt_env_sustain		uint8          // 0-127 
	filt_env_release		uint8          // 0-127 
	mod_env_attack			uint8          // 0-127 
	mod_env_decay_release	uint8          // 0-127
	mod_env_amount			uint8          // 0-127 
	filt_env_amount			uint8          // 0-127 
	filt_frequency1			uint8          // 0-127 
	filt_resonance			uint8          // 0-127 
	filt_frequency2			uint8          // 0-127 
	unison_amount			uint8          // 0-127 
	filt_dist_amount		uint8          // 0-127 
	osc1_sync_tune			uint8          // 0-127 
	osc2_sync_tune			uint8          // 0-127 
	osc1_noise_seed			uint8          // 0-127 
	osc2_noise_seed			uint8          // 0-127 
	osc1_modulator_amount	uint8          // 0-127 In Dual Sine mode
	osc2_modulator_amount	uint8          // 0-127 In Dual Sine mode
	osc2_carrier_pitch		uint8          // 0-127 In Dual Sine mode
	osc2_noise_type			uint8          // 0-127 LP/BP/HP
	osc2_modulator_pitch	uint8          // 0-127 In Dual Sine mode
	osc2_noise_frequency	uint8          // 0-127 
	spare1					uint8          // 0-255 
	spare2					uint8          // 0-255 
	glide_rate				uint8          // 0-127 
	arpeggio_rate			uint8          // 0-127 
	vibrato_rate			uint8          // 0-127 
	vibrato_amount			uint8          // 0-127 
	arpeggio_sync_divisor	uint8          // 0-127 
	lfo1_sync_divisor		uint8          // 0-127 
	lfo2_sync_divisor		uint8          // 0-127 
	transpose				uint8          // 0-127 
	spare3					uint8          // 0-255 
	spare4					uint8          // 0-255 
	osc1_waveform			uint8          // 0-5 
	osc1_sync				bool           // 0-1 
	osc2_waveform			uint8          // 0-5 
	osc2_sync				bool           // 0-1 
	osc2_kbt				bool           // 0-1 Off or On
	osc2_partial			bool           // 0-1 Off or On
	oscmod_type				uint8          // 0-5 
	lfo1_waveform			uint8          // 0-5 
	lfo1_destination		uint8          // 0-11 
	lfo1_env_kbs			uint8          // 0-2
	lfo1_mono				bool           // 0-1 Off or On
	lfo1_invert				bool           // 0-1 Off or On
	lfo2_waveform			uint8          // 0-5 
	lfo2_destination		uint8          // 0-11 
	lfo2_env_kbs			uint8          // 0-2
	lfo2_mono				bool           // 0-1 Off or On
	lfo2_invert				bool           // 0-1 Off or On
	mod_env_invert			bool           // 0-1 Off or On
	mod_env_destination		uint8          // 0-11 
	mod_env_mode			bool           // 0-1 Selects Decay (0) or Release (1) mode
	mod_env_repeat			bool           // 0-1 Off or On
	filt1_type				uint8          // 0-5 
	filt1_slope				uint8          // 0-2 
	filt_env_velocity		bool           // 0-1 Off or On
	filt1_kbt				bool           // 0-1 Off or On
	filt_env_invert			bool           // 0-1 Off or On
	amp_env_exp_attack		bool           // 0-1 Off or On
	mod_env_exp_attack		bool           // 0-1 Off or On
	filt_env_exp_attack		bool           // 0-1 Off or On
	filt_mode				bool           // 0-1 Selects Single or Dual Filter mode
	filt2_env				bool           // 0-1 Selects if Filt2 is controlled by Filt_Env
	filt2_type				uint8          // 0-5 
	filt_bypass				bool           // 0-1 Off or On
	lfo1_clocksync			bool           // 0-1 Off or On
	lfo2_clocksync			bool           // 0-1 Off or On
	arpeggiator_clocksync	bool           // 0-1 Off or On
	oscmix_noise			bool           // 0-1 Off or On
	glide_mode				uint8          // 0-2 Off, On or Auto
	vibrato_source			uint8          // 0-2 Off, Wheel or Aftertouch
	mono_mode				bool           // 0-1 Off or On
	arpeggio_run			bool           // 0-1 Off or On
	spare5					uint8          // 0-255 
	unison_mode				bool           // 0-1 Off or On
	octave_shift			uint8          // 0-4 
	chord_mem_mode			bool           // 0-1 Off or On
	arpeggio_mode			uint8          // 0-3 Up, Down, Up/down or Random
	arpeggio_range			uint8          // 0-3 
	arpeggio_kbd_sync		bool           // 0-1 Off or On
	spare6					uint8          // 0-255 
	spare7					bool           // 0-1 Off or On
	legato_mode				bool           // 0-1 Off or On
	mono_allocation_mode	uint8          // 0-2 Off, Hi or Lo
	wheel_morph_params		MorphParams // 0-127 See ‘Morph parameter list’ below
	a_touch_morph_params	MorphParams // 0-127 See ‘Morph parameter list’ below
	velocity_morph_params	MorphParams // 0-127 See ‘Morph parameter list’ below
	kbd_morph_params		MorphParams // 0-127 See ‘Morph parameter list’ below
	chord_mem_count			uint8          // 0-23	uint8	-24
	chord_mem_position		uint8          // 0-255	uint8	-24
	spare8					uint8          // 0-255 
	checksum				uint8          // 0-255
}

type MorphParams struct {
	// Morph Parameter List (all parameters -128 to 127)
	lfo1_rate 				int8			            
	lfo1_amount 			int8			          
	lfo2_rate 				int8			            
	lfo2_amount 			int8	          
	mod_env_attack 			int8	       
	mod_env_decay_release 	int8	
	mod_env_amount 			int8	       
	osc2_fine_pitch 		int8	      
	osc2_coarse_pitch 		int8	    
	oscmod 					int8	               
	oscmix 					int8	               
	osc1_shape 				int8	           
	osc2_shape 				int8	           
	amp_env_attack 			int8	       
	amp_env_decay 			int8	        
	amp_env_sustain 		int8	      
	amp_env_release 		int8	      
	filt_env_attack 		int8	      
	filt_env_decay 			int8	       
	filt_env_sustain 		int8	     
	filt_env_release 		int8	     
	filt_env_amount 		int8	      
	filt_frequency1 		int8	      
	filt_frequency2 		int8	      
	filt_resonance 			int8	       
	output_level 			int8	  
}
     
type Performance struct {
	version_number			uint16         // Decimal OS version number (# x	100	)
	enabled_slots 			uint8 // 0-15 
	focused_slot 			uint8 // 0-3 
	midi_channel_slot_a 	uint8 // 0-16 0 = Off
	midi_channel_slot_b 	uint8 // 0-16 0 = Off
	midi_channel_slot_c 	uint8 // 0-16 0 = Off
	midi_channel_slot_d 	uint8 // 0-16 0 = Off
	audio_channel_slot_a 	uint8 // 0-5 
	audio_channel_slot_b 	uint8 // 0-5 
	audio_channel_slot_c 	uint8 // 0-5 
	audio_channel_slot_d 	uint8 // 0-5 
	splitpoint_key 			uint8 // 0-127 
	splitpoint 				uint8 // 0-1 Off or On
	sustain_enable 			uint8 // 0-15 
	pitchbend_enable 		uint8 // 0-15 
	modwheel_enable 		uint8 // 0-15 
	bank_slot_a 			uint8 // 0-7 
	program_slot_a 			uint8 // 0-127 
	bank_slot_b 			uint8 // 0-7 
	program_slot_b 			uint8 // 0-127 
	bank_slot_c 			uint8 // 0-7 
	program_slot_c 			uint8 // 0-127 
	bank_slot_d 			uint8 // 0-7 
	program_slot_d 			uint8 // 0-127 
	morph3_source_select 	uint8 // 0-1 Control pedal or aftertouch
	midi_clock_keysync 		uint8 // 0-1 Off or On
	keyboard_hold 			uint8 // 0-1 Off or On
	spare3 					uint8 // Always set to zero
	spare4 					uint8 // Always set to zero
	spare5 					uint8 // Always set to zero
	spare6 					uint8 // Always set to zero
	spare7 					uint8 // Always set to zero
	spare8 					uint8 // Always set to zero
	spare9 					uint8 // Always set to zero
	spare10 				uint8 // Always set to zero
	spare11 				uint8 // Always set to zero
	spare12 				uint8 // Always set to zero
	midi_clock_rate 		uint16 // 0-210 
	bend_range_up 			uint8 // 0-24 
	bend_range_down 		uint8 // 0-24 
	patchname_slot_a 		[16]uint8 // Max 16 characters long
	patchname_slot_b 		[16]uint8 // Max 16 characters long
	patchname_slot_c 		[16]uint8 // Max 16 characters long
	patchname_slot_d 		[16]uint8 // Max 16 characters long
	patch_data 				[4]Program
	checksum 				uint8 
}

func ParseSysex(sysex []byte, memory *PatchMemory) {
	// Read header to catalogue and identify type
	messageType := sysex[3]
	bank 		:= sysex[4]
	location 	:= sysex[5]
	name 		:= sysex[6:22]
	
	switch messageType {
	case ProgramFromSlot, ProgramFromMemory:
	  ParseProgramSysex(sysex, memory, bank, location, name)
	case PerformanceFromSlot, PerformanceFromMemory:
	  ParsePerformanceSysex(sysex, memory, bank, location, name)
	default: 
	  fmt.Printf("Skipping non-patch sysex (type %#x - %v)\n", messageType, messageType)
	}
}

func ParsePerformanceSysex(sysex []byte, memory *PatchMemory, bank uint8, location uint8, name []byte) {
	printableName			:= fmt.Sprintf("%-16s", strings.TrimRight(string(name), "\x00"))

	version 				:= float64(uint16(sysex[VersionOffset]) << 8 + uint16(sysex[VersionOffset + 1])) / 100.0
	performanceSysex 		:= sysex[PatchDataOffset:len(sysex) - 1] // strip trailing F7
	performanceBitstream 	:= decodeSysexToBitstream(performanceSysex)

	if !checksumValid(performanceBitstream) {
		fmt.Printf("Found INVALID performance: (%v:%03d) %q v%1.2f\n", bank, location, printableName, version)
		return
	}

	fmt.Printf("Found Performance: (%v:%03d) %-16.16q v%1.2f\n", bank, location, printableName, version)
}

func ParseProgramSysex(sysex []byte, memory *PatchMemory, bank uint8, location uint8, name []byte) {
	printableName		:= fmt.Sprintf("%-16s", strings.TrimRight(string(name), "\x00"))

	version 			:= float64(uint16(sysex[VersionOffset]) << 8 + uint16(sysex[VersionOffset + 1])) / 100.0
	programSysex 		:= sysex[PatchDataOffset:len(sysex) - 1] // strip trailing F7
	programBitstream 	:= decodeSysexToBitstream(programSysex)

	if !checksumValid(programBitstream) {
		fmt.Printf("Found INVALID program: (%v:%03d) %q v%1.2f\n", bank, location, printableName, version)
		return
	}

	fmt.Printf("Found Program: (%v:%03d) %-16.16q v%1.2f\n", bank, location, printableName, version)
}

func decodeSysexToBitstream(payload []byte) []byte {
	// MIDI 8-bit to bitstream decoding
	// Every byte of the MIDI stream is actually only 7 bits of the payload bitstream
	// so we need to drop a bit every byte and re-concatenate the bits
	
	buf    := bytes.NewBuffer(nil)
	reader := bitstream.NewReader(strings.NewReader(string(payload)))
	writer := bitstream.NewWriter(buf)
	i      := 0

	for {
		bit, err := reader.ReadBit()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(fmt.Sprintf("GetBit returned error err %v", err.Error()))
		}
		if i % 8 == 0 {
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

func checksumValid(decodedBitstream []byte) bool {
	var runningSum uint8
	var sizeCorrect bool
	
	if len(decodedBitstream) == 191 || len(decodedBitstream) == 859 {
		sizeCorrect = true
	}

	checksum := decodedBitstream[len(decodedBitstream) - 1]
	payload  := decodedBitstream[:len(decodedBitstream) - 1]
	
	for _, currByte := range(payload) {
		runningSum += uint8(currByte)
	}

	checksumCorrect := checksum == runningSum

	return sizeCorrect && checksumCorrect
}