package sf2

import (
	"fmt"
	"sort"
)

// Selector identifies a preset by its MIDI bank and program.
type Selector struct {
	Bank    uint16
	Program uint16
}

// guardFrames is the run of zero sample-points the SF2 spec recommends after each sample so a
// synth can interpolate past the loop without reading a neighbour.
const guardFrames = 46

// stereoMask picks the right/left/linked bits of a sample's type. Only for those is SampleLink a
// real reference; a mono sample leaves the field zero, which must not be read as a link to sample 0.
const stereoMask = 2 | 4 | 8

// Subset returns a new SoundFont holding only the presets in keep (matched by bank+program) plus
// the instruments and samples they reference. Bank and program numbers are preserved, so a caller
// that loads a patch by program number needs no change. Stereo partners are pulled in via the
// sample links.
func (sf *SF2) Subset(keep []Selector) (*SF2, error) {
	want := make(map[Selector]bool, len(keep))
	for _, s := range keep {
		want[s] = true
	}

	var keptPresets []int
	for i := 0; i+1 < len(sf.Phdr); i++ { // the last phdr entry is the terminal
		if want[Selector{Bank: sf.Phdr[i].Bank, Program: sf.Phdr[i].Preset}] {
			keptPresets = append(keptPresets, i)
		}
	}
	if len(keptPresets) == 0 {
		return nil, fmt.Errorf("no presets matched the %d selector(s)", len(keep))
	}

	instSet := map[uint16]bool{}
	for _, pi := range keptPresets {
		for _, g := range sf.presetGens(pi) {
			if g.Oper == genInstrument {
				instSet[g.Amount] = true
			}
		}
	}

	smplSet := map[uint16]bool{}
	for inst := range instSet {
		for _, g := range sf.instGens(int(inst)) {
			if g.Oper == genSampleID {
				smplSet[g.Amount] = true
			}
		}
	}
	for s := range smplSet { // keep a stereo sample's partner too
		if int(s) < len(sf.Shdr)-1 {
			h := sf.Shdr[s]
			if h.SampleType&stereoMask != 0 && int(h.SampleLink) < len(sf.Shdr)-1 {
				smplSet[h.SampleLink] = true
			}
		}
	}

	keptInsts := sortedKeys(instSet)
	keptSamples := sortedKeys(smplSet)
	instRemap := remap(keptInsts)
	smplRemap := remap(keptSamples)

	out := &SF2{Info: append([]byte(nil), sf.Info...)}
	sf.buildSamples(out, keptSamples)
	sf.buildInstruments(out, keptInsts, smplRemap)
	sf.buildPresets(out, keptPresets, instRemap)
	return out, nil
}

// presetGens is every generator across all zones of preset pi, for reference collection.
func (sf *SF2) presetGens(pi int) []Generator {
	from := sf.Pbag[sf.Phdr[pi].BagIndex].GenIndex
	to := sf.Pbag[sf.Phdr[pi+1].BagIndex].GenIndex
	return sf.Pgen[from:to]
}

func (sf *SF2) instGens(ii int) []Generator {
	from := sf.Ibag[sf.Inst[ii].BagIndex].GenIndex
	to := sf.Ibag[sf.Inst[ii+1].BagIndex].GenIndex
	return sf.Igen[from:to]
}

func (sf *SF2) buildSamples(out *SF2, kept []uint16) {
	for _, si := range kept {
		h := sf.Shdr[si]
		data := sf.sampleData(h.Start, h.End)
		newStart := uint32(len(out.Smpl) / 2)
		out.Smpl = append(out.Smpl, data...)
		out.Smpl = append(out.Smpl, make([]byte, guardFrames*2)...)

		nh := h
		nh.Start = newStart
		nh.End = newStart + (h.End - h.Start)
		nh.StartLoop = newStart + (h.StartLoop - h.Start)
		nh.EndLoop = newStart + (h.EndLoop - h.Start)
		nh.SampleLink = 0 // fixed up below
		out.Shdr = append(out.Shdr, nh)
	}
	// Remap stereo links now that every kept sample has a new index. Mono samples keep the
	// zeroed link set above.
	remapS := remap(kept)
	for i, si := range kept {
		h := sf.Shdr[si]
		if h.SampleType&stereoMask == 0 {
			continue
		}
		if nl, ok := remapS[h.SampleLink]; ok {
			out.Shdr[i].SampleLink = nl
		}
	}
	out.Shdr = append(out.Shdr, SampleHeader{Name: name20("EOS")})
}

func (sf *SF2) buildInstruments(out *SF2, kept []uint16, smplRemap map[uint16]uint16) {
	for _, ii := range kept {
		h := sf.Inst[ii]
		h.BagIndex = uint16(len(out.Ibag))
		out.Inst = append(out.Inst, h)
		for bi := sf.Inst[ii].BagIndex; bi < sf.Inst[ii+1].BagIndex; bi++ {
			out.Ibag = append(out.Ibag, Bag{GenIndex: uint16(len(out.Igen)), ModIndex: uint16(len(out.Imod))})
			for _, g := range sf.Igen[sf.Ibag[bi].GenIndex:sf.Ibag[bi+1].GenIndex] {
				if g.Oper == genSampleID {
					g.Amount = smplRemap[g.Amount]
				}
				out.Igen = append(out.Igen, g)
			}
			out.Imod = append(out.Imod, sf.Imod[sf.Ibag[bi].ModIndex:sf.Ibag[bi+1].ModIndex]...)
		}
	}
	out.Igen = append(out.Igen, Generator{})
	out.Imod = append(out.Imod, Modulator{})
	out.Ibag = append(out.Ibag, Bag{GenIndex: uint16(len(out.Igen) - 1), ModIndex: uint16(len(out.Imod) - 1)})
	out.Inst = append(out.Inst, InstHeader{Name: name20("EOI"), BagIndex: uint16(len(out.Ibag) - 1)})
}

func (sf *SF2) buildPresets(out *SF2, kept []int, instRemap map[uint16]uint16) {
	for _, pi := range kept {
		h := sf.Phdr[pi]
		h.BagIndex = uint16(len(out.Pbag))
		out.Phdr = append(out.Phdr, h)
		for bi := sf.Phdr[pi].BagIndex; bi < sf.Phdr[pi+1].BagIndex; bi++ {
			out.Pbag = append(out.Pbag, Bag{GenIndex: uint16(len(out.Pgen)), ModIndex: uint16(len(out.Pmod))})
			for _, g := range sf.Pgen[sf.Pbag[bi].GenIndex:sf.Pbag[bi+1].GenIndex] {
				if g.Oper == genInstrument {
					g.Amount = instRemap[g.Amount]
				}
				out.Pgen = append(out.Pgen, g)
			}
			out.Pmod = append(out.Pmod, sf.Pmod[sf.Pbag[bi].ModIndex:sf.Pbag[bi+1].ModIndex]...)
		}
	}
	out.Pgen = append(out.Pgen, Generator{})
	out.Pmod = append(out.Pmod, Modulator{})
	out.Pbag = append(out.Pbag, Bag{GenIndex: uint16(len(out.Pgen) - 1), ModIndex: uint16(len(out.Pmod) - 1)})
	out.Phdr = append(out.Phdr, PresetHeader{Name: name20("EOP"), BagIndex: uint16(len(out.Pbag) - 1)})
}

func (sf *SF2) sampleData(start, end uint32) []byte {
	lo, hi := int(start)*2, int(end)*2
	if lo > len(sf.Smpl) {
		lo = len(sf.Smpl)
	}
	if hi > len(sf.Smpl) {
		hi = len(sf.Smpl)
	}
	if lo > hi {
		lo = hi
	}
	return append([]byte(nil), sf.Smpl[lo:hi]...)
}

func sortedKeys(set map[uint16]bool) []uint16 {
	out := make([]uint16, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func remap(kept []uint16) map[uint16]uint16 {
	m := make(map[uint16]uint16, len(kept))
	for newIdx, oldIdx := range kept {
		m[oldIdx] = uint16(newIdx)
	}
	return m
}

func name20(s string) [20]byte {
	var a [20]byte
	copy(a[:], s)
	return a
}
