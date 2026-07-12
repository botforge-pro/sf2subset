package sf2

import (
	"reflect"
	"testing"
)

// fixture builds a tiny but structurally complete SoundFont: two presets (Piano at program 0,
// Organ at program 18), each pointing at one instrument, each instrument at one sample.
func fixture() *SF2 {
	info := append(chunk("ifil", []byte{2, 0, 1, 0}), chunk("INAM", []byte("Test\x00\x00"))...)
	return &SF2{
		Info: info,
		Smpl: []byte{1, 0, 2, 0, 3, 0, 4, 0, 0, 0, 0, 0},
		Phdr: []PresetHeader{
			{Name: name20("Piano"), Preset: 0, Bank: 0, BagIndex: 0},
			{Name: name20("Organ"), Preset: 18, Bank: 0, BagIndex: 1},
			{Name: name20("EOP"), BagIndex: 2},
		},
		Pbag: []Bag{{GenIndex: 0}, {GenIndex: 1}, {GenIndex: 2}},
		Pmod: []Modulator{{}},
		Pgen: []Generator{
			{Oper: genInstrument, Amount: 0},
			{Oper: genInstrument, Amount: 1},
			{},
		},
		Inst: []InstHeader{
			{Name: name20("PianoInst"), BagIndex: 0},
			{Name: name20("OrganInst"), BagIndex: 1},
			{Name: name20("EOI"), BagIndex: 2},
		},
		Ibag: []Bag{{GenIndex: 0}, {GenIndex: 1}, {GenIndex: 2}},
		Imod: []Modulator{{}},
		Igen: []Generator{
			{Oper: genSampleID, Amount: 0},
			{Oper: genSampleID, Amount: 1},
			{},
		},
		Shdr: []SampleHeader{
			{Name: name20("PianoSmpl"), Start: 0, End: 2, StartLoop: 0, EndLoop: 2, SampleRate: 44100, SampleType: 1},
			{Name: name20("OrganSmpl"), Start: 2, End: 4, StartLoop: 2, EndLoop: 4, SampleRate: 44100, SampleType: 1},
			{Name: name20("EOS")},
		},
	}
}

func TestRoundTrip(t *testing.T) {
	orig := fixture()
	parsed, err := Parse(orig.Bytes())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(orig, parsed) {
		t.Errorf("round-trip mismatch\n orig=%+v\n got =%+v", orig, parsed)
	}
}

func TestParseRejectsNonSF2(t *testing.T) {
	if _, err := Parse([]byte("not a soundfont")); err == nil {
		t.Error("expected error for non-sf2 input")
	}
}
