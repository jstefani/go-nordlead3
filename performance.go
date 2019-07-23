package nordlead3

import (
	"errors"
)

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

func (performance *Performance) dumpSysex() (*[]byte, error) {
	if performance == nil {
		return nil, errors.New("Cannot dump a blank performance - no init values set!")
	}

	payload, err := bitstreamFromStruct(performance)
	if err != nil {
		return nil, err
	}

	payload = append(payload, checksum8(payload))
	packedPayload := packSysex(payload)

	return &packedPayload, nil
}
