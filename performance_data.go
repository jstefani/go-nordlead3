package nordlead3

import (
	"errors"
)

// Mask values such as Enabled_slots and Sustain_enable have 1 = slot 1, 2 = slot 2, 4 = slot 3, 8 = slot 4.
type PerformanceData struct {
	Version_number       uint        `len:"16"`                  // Decimal OS version number (# x	100	)
	Enabled_slots        uint        `len:"8" min:"0" max:"127"` // 0-15
	Focused_slot         uint        `len:"8" min:"0" max:"127"` // 0-3
	Midi_channel_slot_a  uint        `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Midi_channel_slot_b  uint        `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Midi_channel_slot_c  uint        `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Midi_channel_slot_d  uint        `len:"8" min:"0" max:"127"` // 0-16 0 = Off
	Audio_channel_slot_a uint        `len:"8" min:"0" max:"127"` // 0-5
	Audio_channel_slot_b uint        `len:"8" min:"0" max:"127"` // 0-5
	Audio_channel_slot_c uint        `len:"8" min:"0" max:"127"` // 0-5
	Audio_channel_slot_d uint        `len:"8" min:"0" max:"127"` // 0-5
	Splitpoint_key       uint        `len:"8" min:"0" max:"127"` // 0-127
	Spare1               uint        `len:"7"`                   // Really just padding for the bool splitpoint_enable
	Splitpoint_enable    bool        `len:"1"`                   // 0-1 Off or On
	Sustain_enable       uint        `len:"8" min:"0" max:"127"` // 0-15 is a mask 0b00001111 for each of the four slots
	Pitchbend_enable     uint        `len:"8" min:"0" max:"127"` // 0-15 is a mask 0b00001111 for each of the four slots
	Modwheel_enable      uint        `len:"8" min:"0" max:"127"` // 0-15 is a mask 0b00001111 for each of the four slots
	Bank_slot_a          uint        `len:"8" min:"0" max:"7"`
	Program_slot_a       uint        `len:"8" min:"0" max:"127"`
	Bank_slot_b          uint        `len:"8" min:"0" max:"7"`
	Program_slot_b       uint        `len:"8" min:"0" max:"127"`
	Bank_slot_c          uint        `len:"8" min:"0" max:"7"`
	Program_slot_c       uint        `len:"8" min:"0" max:"127"`
	Bank_slot_d          uint        `len:"8" min:"0" max:"7"`
	Program_slot_d       uint        `len:"8" min:"0" max:"127"`
	Spare2               uint        `len:"7"` // padding
	Morph3_source_select bool        `len:"1"` // 0-1 Control pedal or aftertouch
	Spare3               uint        `len:"7"` // padding
	Midi_clock_keysync   bool        `len:"1"`
	Spare4               uint        `len:"7"` // padding
	Keyboard_hold        bool        `len:"1"`
	Spare5               uint        `len:"8"`
	Spare6               uint        `len:"8"`
	Spare7               uint        `len:"8"`
	Spare8               uint        `len:"8"`
	Spare9               uint        `len:"8"`
	Spare10              uint        `len:"8"`
	Spare11              uint        `len:"8"`
	Spare12              uint        `len:"8"`
	Spare13              uint        `len:"8"`
	Spare14              uint        `len:"8"`
	Spare15              uint        `len:"8"`
	Midi_clock_rate      uint        `len:"8" min:"0" max:"210"` // 0-210
	Bend_range_up        uint        `len:"8" min:"0" max:"24"`  // 0-24
	Bend_range_down      uint        `len:"8" min:"0" max:"24"`  // 0-24
	Patchname_slot_a     [16]byte    `len:"8"`                   // Offset 42
	Patchname_slot_b     [16]byte    `len:"8"`
	Patchname_slot_c     [16]byte    `len:"8"`
	Patchname_slot_d     [16]byte    `len:"8"`
	Patch_data_a         ProgramData `len:"1504"`
	Patch_data_b         ProgramData `len:"1504"`
	Patch_data_c         ProgramData `len:"1504"`
	Patch_data_d         ProgramData `len:"1504"`
	Checksum             uint        `len:"8"`
}

func (performanceData *PerformanceData) dumpSysex() (*[]byte, error) {
	if performanceData == nil {
		return nil, errors.New("Cannot dump a blank performance - no init values set!")
	}

	payload, err := bitstreamFromStruct(performanceData)
	if err != nil {
		return nil, err
	}

	payload = append(payload, checksum8(payload))
	packedPayload := packSysex(payload)

	return &packedPayload, nil
}

// Requires a properly formatted bitstream decoded from NL3 sysex
func newPerformanceFromBitstream(data []byte) (*PerformanceData, error) {
	performanceData := new(PerformanceData)
	err := populateStructFromBitstream(performanceData, data)
	return performanceData, err
}
