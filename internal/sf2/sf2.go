// Package sf2 reads, writes and subsets SoundFont 2 (.sf2) files. A SoundFont is a RIFF file
// (chunk id, little-endian size, payload) with three top-level LISTs: INFO (metadata), sdta
// (raw sample data) and pdta (the preset/instrument/sample tables). Only the pdta tables and
// the sample data matter for subsetting; INFO is copied verbatim.
package sf2

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Fixed-size pdta records, laid out exactly as in the SF2 spec (packed, little-endian). Every
// table ends with a terminal sentinel record, which the parser keeps and the writer reproduces.

// PresetHeader is a phdr record (38 bytes). Preset and Bank are the MIDI program and bank.
type PresetHeader struct {
	Name       [20]byte
	Preset     uint16
	Bank       uint16
	BagIndex   uint16
	Library    uint32
	Genre      uint32
	Morphology uint32
}

// Bag is a pbag or ibag record (4 bytes): the start index of this zone's generators and
// modulators.
type Bag struct {
	GenIndex uint16
	ModIndex uint16
}

// Modulator is a pmod or imod record (10 bytes). Kept verbatim.
type Modulator struct {
	SrcOper    uint16
	DestOper   uint16
	Amount     int16
	AmtSrcOper uint16
	TransOper  uint16
}

// Generator is a pgen or igen record (4 bytes). Amount is a union; only two opers are read as
// indices: Instrument (41) in a preset generator and SampleID (53) in an instrument generator.
type Generator struct {
	Oper   uint16
	Amount uint16
}

const (
	genInstrument = 41
	genSampleID   = 53
)

// InstHeader is an inst record (22 bytes).
type InstHeader struct {
	Name     [20]byte
	BagIndex uint16
}

// SampleHeader is a shdr record (46 bytes). Start/End and the loop points are sample-frame
// offsets into the smpl data.
type SampleHeader struct {
	Name            [20]byte
	Start           uint32
	End             uint32
	StartLoop       uint32
	EndLoop         uint32
	SampleRate      uint32
	OriginalPitch   uint8
	PitchCorrection int8
	SampleLink      uint16
	SampleType      uint16
}

// SF2 is a parsed SoundFont: INFO copied verbatim, the 16-bit sample data, and the nine pdta
// tables (each still carrying its terminal record).
type SF2 struct {
	Info []byte // raw INFO sub-chunks
	Smpl []byte // 16-bit little-endian sample data

	Phdr []PresetHeader
	Pbag []Bag
	Pmod []Modulator
	Pgen []Generator
	Inst []InstHeader
	Ibag []Bag
	Imod []Modulator
	Igen []Generator
	Shdr []SampleHeader
}

// Parse reads a SoundFont from raw bytes.
func Parse(data []byte) (*SF2, error) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "sfbk" {
		return nil, fmt.Errorf("not a RIFF/sfbk file")
	}
	sf := &SF2{}
	body := data[12:]
	for len(body) >= 8 {
		id := string(body[0:4])
		size := int(binary.LittleEndian.Uint32(body[4:8]))
		if id != "LIST" || 8+size > len(body) {
			return nil, fmt.Errorf("malformed top-level chunk %q", id)
		}
		listType := string(body[8:12])
		payload := body[12 : 8+size]
		if err := sf.parseList(listType, payload); err != nil {
			return nil, err
		}
		advance := 8 + size
		if advance%2 == 1 {
			advance++ // chunks are word-aligned
		}
		if advance > len(body) {
			break
		}
		body = body[advance:]
	}
	return sf, nil
}

func (sf *SF2) parseList(listType string, payload []byte) error {
	if listType == "INFO" {
		sf.Info = append([]byte(nil), payload...) // copied verbatim, sub-chunks untouched
		return nil
	}
	for len(payload) >= 8 {
		id := string(payload[0:4])
		size := int(binary.LittleEndian.Uint32(payload[4:8]))
		if 8+size > len(payload) {
			return fmt.Errorf("sub-chunk %q overruns %s", id, listType)
		}
		data := payload[8 : 8+size]
		if err := sf.parseSubChunk(id, data); err != nil {
			return err
		}
		advance := 8 + size
		if advance%2 == 1 {
			advance++
		}
		if advance > len(payload) {
			break
		}
		payload = payload[advance:]
	}
	return nil
}

func (sf *SF2) parseSubChunk(id string, data []byte) error {
	switch id {
	case "smpl":
		sf.Smpl = append([]byte(nil), data...)
	case "phdr":
		return readRecords(data, &sf.Phdr)
	case "pbag":
		return readRecords(data, &sf.Pbag)
	case "pmod":
		return readRecords(data, &sf.Pmod)
	case "pgen":
		return readRecords(data, &sf.Pgen)
	case "inst":
		return readRecords(data, &sf.Inst)
	case "ibag":
		return readRecords(data, &sf.Ibag)
	case "imod":
		return readRecords(data, &sf.Imod)
	case "igen":
		return readRecords(data, &sf.Igen)
	case "shdr":
		return readRecords(data, &sf.Shdr)
	}
	// The INFO list's own sub-chunks (ifil, INAM, ...) are captured as raw bytes by the caller.
	return nil
}

func readRecords[T any](data []byte, out *[]T) error {
	var zero T
	recSize := binary.Size(zero)
	if recSize <= 0 || len(data)%recSize != 0 {
		return fmt.Errorf("record data (%d) not a multiple of %d", len(data), recSize)
	}
	n := len(data) / recSize
	recs := make([]T, n)
	if err := binary.Read(bytes.NewReader(data), binary.LittleEndian, recs); err != nil {
		return err
	}
	*out = recs
	return nil
}

// Bytes serialises the SoundFont back to raw bytes.
func (sf *SF2) Bytes() []byte {
	info := chunkRaw("LIST", append([]byte("INFO"), sf.Info...))

	var sdtaBody bytes.Buffer
	sdtaBody.WriteString("sdta")
	sdtaBody.Write(chunk("smpl", sf.Smpl))
	sdta := chunkRaw("LIST", sdtaBody.Bytes())

	var pdtaBody bytes.Buffer
	pdtaBody.WriteString("pdta")
	pdtaBody.Write(chunk("phdr", recordBytes(sf.Phdr)))
	pdtaBody.Write(chunk("pbag", recordBytes(sf.Pbag)))
	pdtaBody.Write(chunk("pmod", recordBytes(sf.Pmod)))
	pdtaBody.Write(chunk("pgen", recordBytes(sf.Pgen)))
	pdtaBody.Write(chunk("inst", recordBytes(sf.Inst)))
	pdtaBody.Write(chunk("ibag", recordBytes(sf.Ibag)))
	pdtaBody.Write(chunk("imod", recordBytes(sf.Imod)))
	pdtaBody.Write(chunk("igen", recordBytes(sf.Igen)))
	pdtaBody.Write(chunk("shdr", recordBytes(sf.Shdr)))
	pdta := chunkRaw("LIST", pdtaBody.Bytes())

	var body bytes.Buffer
	body.WriteString("sfbk")
	body.Write(info)
	body.Write(sdta)
	body.Write(pdta)
	return chunkRaw("RIFF", body.Bytes())
}

// chunk wraps payload as id + size + payload, padded to an even length.
func chunk(id string, payload []byte) []byte {
	return chunkRaw(id, payload)
}

func chunkRaw(id string, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString(id)
	_ = binary.Write(&b, binary.LittleEndian, uint32(len(payload)))
	b.Write(payload)
	if len(payload)%2 == 1 {
		b.WriteByte(0)
	}
	return b.Bytes()
}

func recordBytes[T any](recs []T) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.LittleEndian, recs)
	return b.Bytes()
}
