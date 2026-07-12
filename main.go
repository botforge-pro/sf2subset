// Command sf2subset trims a SoundFont (.sf2) down to a chosen set of presets, keeping the
// instruments and samples they use and preserving bank/program numbers.
//
//	sf2subset -i GeneralUser-GS.sf2 -o small.sf2 -p 0,18,81
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/botforge-pro/sf2subset/internal/sf2"
)

func main() {
	in := flag.String("i", "", "input .sf2 file")
	out := flag.String("o", "", "output .sf2 file")
	presets := flag.String("p", "", "presets to keep: comma-separated bank:program (bank optional, defaults to 0), e.g. 0,18,81 or 0:0,0:18")
	flag.Parse()

	if *in == "" || *out == "" || *presets == "" {
		flag.Usage()
		os.Exit(2)
	}

	sel, err := parseSelectors(*presets)
	if err != nil {
		fatal(err)
	}

	data, err := os.ReadFile(*in)
	if err != nil {
		fatal(err)
	}
	src, err := sf2.Parse(data)
	if err != nil {
		fatal(err)
	}
	dst, err := src.Subset(sel)
	if err != nil {
		fatal(err)
	}
	result := dst.Bytes()
	if err := os.WriteFile(*out, result, 0o644); err != nil {
		fatal(err)
	}

	fmt.Printf("%s: %d preset(s), %d instrument(s), %d sample(s)\n",
		*out, len(dst.Phdr)-1, len(dst.Inst)-1, len(dst.Shdr)-1)
	fmt.Printf("size: %d -> %d bytes (%.1f%% of original)\n",
		len(data), len(result), 100*float64(len(result))/float64(len(data)))
}

// parseSelectors reads "bank:program" pairs (or a bare "program", bank 0) separated by commas.
func parseSelectors(s string) ([]sf2.Selector, error) {
	var out []sf2.Selector
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var bank, prog uint16
		if i := strings.IndexByte(part, ':'); i >= 0 {
			b, err := strconv.ParseUint(strings.TrimSpace(part[:i]), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("bad bank in %q: %w", part, err)
			}
			p, err := strconv.ParseUint(strings.TrimSpace(part[i+1:]), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("bad program in %q: %w", part, err)
			}
			bank, prog = uint16(b), uint16(p)
		} else {
			p, err := strconv.ParseUint(part, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("bad program in %q: %w", part, err)
			}
			prog = uint16(p)
		}
		out = append(out, sf2.Selector{Bank: bank, Program: prog})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no presets given")
	}
	return out, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "sf2subset:", err)
	os.Exit(1)
}
