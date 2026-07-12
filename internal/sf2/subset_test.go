package sf2

import (
	"reflect"
	"strings"
	"testing"
)

func nameStr(a [20]byte) string {
	return strings.TrimRight(string(a[:]), "\x00")
}

// keeping one preset drops the other preset, its instrument and its sample, and renumbers the
// surviving references to the compacted tables.
func TestSubsetKeepsOnlySelectedPreset(t *testing.T) {
	got, err := fixture().Subset([]Selector{{Bank: 0, Program: 18}}) // Organ only
	if err != nil {
		t.Fatalf("subset: %v", err)
	}

	if len(got.Phdr) != 2 { // one real preset + terminal
		t.Fatalf("phdr = %d entries, want 2", len(got.Phdr))
	}
	if p := got.Phdr[0]; p.Preset != 18 || p.Bank != 0 || nameStr(p.Name) != "Organ" {
		t.Errorf("kept preset = %q prog %d bank %d, want Organ/18/0", nameStr(p.Name), p.Preset, p.Bank)
	}
	if len(got.Inst) != 2 || nameStr(got.Inst[0].Name) != "OrganInst" {
		t.Errorf("inst = %d entries, first %q; want 2 / OrganInst", len(got.Inst), nameStr(got.Inst[0].Name))
	}
	if len(got.Shdr) != 2 || nameStr(got.Shdr[0].Name) != "OrganSmpl" {
		t.Errorf("shdr = %d entries, first %q; want 2 / OrganSmpl", len(got.Shdr), nameStr(got.Shdr[0].Name))
	}

	// The instrument generator must now point at the single remaining instrument (index 0),
	// and the sample generator at the single remaining sample (index 0).
	if g := got.Pgen[0]; g.Oper != genInstrument || g.Amount != 0 {
		t.Errorf("preset gen = oper %d amount %d, want instrument->0", g.Oper, g.Amount)
	}
	if g := got.Igen[0]; g.Oper != genSampleID || g.Amount != 0 {
		t.Errorf("inst gen = oper %d amount %d, want sampleID->0", g.Oper, g.Amount)
	}

	// Sample data is the organ's two frames, relocated to offset 0, followed by the guard.
	s := got.Shdr[0]
	if s.Start != 0 || s.End != 2 {
		t.Errorf("relocated sample = start %d end %d, want 0/2", s.Start, s.End)
	}
	if want := []byte{3, 0, 4, 0}; !reflect.DeepEqual(got.Smpl[:4], want) {
		t.Errorf("sample data = %v, want %v", got.Smpl[:4], want)
	}
	if len(got.Smpl) != 4+guardFrames*2 {
		t.Errorf("smpl len = %d, want %d", len(got.Smpl), 4+guardFrames*2)
	}
}

func TestSubsetOutputRoundTrips(t *testing.T) {
	got, err := fixture().Subset([]Selector{{Bank: 0, Program: 18}})
	if err != nil {
		t.Fatalf("subset: %v", err)
	}
	parsed, err := Parse(got.Bytes())
	if err != nil {
		t.Fatalf("parse subset: %v", err)
	}
	if !reflect.DeepEqual(got, parsed) {
		t.Errorf("subset does not round-trip\n got   =%+v\n parsed=%+v", got, parsed)
	}
}

func TestSubsetKeepingAllPreservesBothPresets(t *testing.T) {
	got, err := fixture().Subset([]Selector{{Bank: 0, Program: 0}, {Bank: 0, Program: 18}})
	if err != nil {
		t.Fatalf("subset: %v", err)
	}
	if len(got.Phdr) != 3 || len(got.Inst) != 3 || len(got.Shdr) != 3 {
		t.Errorf("keeping all: phdr %d inst %d shdr %d, want 3/3/3", len(got.Phdr), len(got.Inst), len(got.Shdr))
	}
}

func TestSubsetNoMatchErrors(t *testing.T) {
	if _, err := fixture().Subset([]Selector{{Bank: 0, Program: 99}}); err == nil {
		t.Error("expected error when no preset matches")
	}
}
