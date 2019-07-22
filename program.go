package nordlead3

import (
	"errors"
)

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

func (program *Program) DumpSysex() (*[]byte, error) {
	if program == nil {
		return nil, errors.New("Cannot dump a blank program - no init values set!")
	}

	payload, err := bitstreamFromStruct(program)
	if err != nil {
		return nil, err
	}

	payload = append(payload, checksum8(payload))

	packedPayload := packSysex(payload)

	return &packedPayload, nil
}
