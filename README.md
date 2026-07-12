# sf2subset

Trim a [SoundFont 2](https://en.wikipedia.org/wiki/SoundFont) (`.sf2`) down to a chosen set of
presets, keeping only the instruments and samples those presets use. Bank and program numbers are
preserved, so code that loads a patch by program number needs no change.

## Why

`AVAudioUnitSampler` (and most embedded synths) load `.sf2` but not the Ogg-compressed `.sf3`, so a
general-MIDI bank like [GeneralUser GS](https://schristiancollins.com/generaluser.php) ships as a
~31 MB file even when an app plays three patches. This tool cuts it to just those patches: for
24-TET Keys (piano/organ/synth) that is a **7.5 %** file.

Polyphone can do the same by hand in its GUI, but this is a scriptable CLI so shrinking stays part
of the build and survives a change of instrument set.

## Usage

```sh
sf2subset -i GeneralUser-GS.sf2 -o small.sf2 -p 0,18,81
```

- `-i` input `.sf2`
- `-o` output `.sf2`
- `-p` presets to keep: comma-separated `bank:program` (bank optional, defaults to `0`), e.g.
  `0,18,81` or `0:0,0:18,0:81`

It prints the kept preset/instrument/sample counts and the before/after size.

## How it works

A SoundFont is a RIFF file with preset, instrument and sample tables that reference each other by
index. `sf2subset` keeps the selected presets, walks their generators to the instruments they use,
walks those to the samples they use (pulling in stereo partners via the sample links), then rebuilds
every table with compacted indices and relocated sample data. Modulators and generators are copied
verbatim, so the sound is unchanged.

## Develop

```sh
make test   # go test ./...
make build  # go build -o sf2subset .
make check  # gofmt check + go vet + tests
```

The `internal/sf2` package parses, writes and subsets; it is covered by round-trip and subset tests
built on a small synthetic SoundFont, so the logic is verified without a large binary fixture.
