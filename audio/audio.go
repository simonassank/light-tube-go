package audio

import (
    "github.com/cocoonlife/goalsa"
    "github.com/simonassank/aubio-go"
    "fmt"
    "flag"
)

var (
    SrcPath     = flag.String("src", "", "Path to source file. Required")
    Samplerate  = flag.Int("samplerate", 44100, "Sample rate to use for the audio file")
    Blocksize   = flag.Int("blocksize", 512, "Blocksize use for the audio file")
    Bufsize     = flag.Int("bufsize", 1024, "Bufsize use for the audio file")
    Periods     = flag.Int("periods", 1, "Periods")
    PerFrames   = flag.Int("perframes", 1, "Period frames")
)

var (
    pitch       *aubio.Pitch
    phVoc       *aubio.PhaseVoc
    fb          *aubio.FilterBank
    tempo       *aubio.Tempo
    IsReady     = false
    BufferId    = uint64(0)
    WriterId    = uint64(0)
)

func AudioPassThrough(bufferChannel chan []float64, playbackChannel chan uint64) {
    c, p := NewAudio()
    IsReady = true
    buff := make([]float64, *Bufsize)
    timeline := [][]float64{}
    isWrite := false
    for {
        if (!isWrite || len(timeline) < 100) {
            BufferId += 1
            samples, err := c.Read(buff)
            if (samples == 0) {
                break
            }
            if (err != nil) {
                break
            }
            buffCopy := make([]float64, *Bufsize)
            copy(buffCopy, buff)
            timeline = append(timeline, buffCopy)
            bufferChannel <- buff
            isWrite = true
        } else {
            if (WriterId == 0) {
                playbackChannel <- 0
            }
            WriterId += 1
            oldBuff := timeline[0]
            timeline = timeline[1:]
            _, err := p.Write(oldBuff)
            if (err != nil) {
                fmt.Println(err)
            }

            isWrite = false
        }
    }
}

func NewAudio() (c *alsa.CaptureDevice, p *alsa.PlaybackDevice) {
    c, errC := alsa.NewCaptureDevice(
        "plughw:CARD=Device,DEV=0",
        2,
        alsa.FormatFloat64LE,
        *Samplerate,
        alsa.BufferParams{
            *Samplerate,
            *PerFrames,
            *Periods,
         },
    )

    fmt.Println("Capure err", errC)

    p, errP := alsa.NewPlaybackDevice(
        "plughw:CARD=Device,DEV=0",
        2,
        alsa.FormatFloat64LE,
        *Samplerate,
        alsa.BufferParams{
            *Samplerate,
            *PerFrames,
            *Periods,
        },
    )

    fmt.Println("Playback err", errP)
    fmt.Println("filters start")
    pitch, phVoc, fb = getFilters()
    fmt.Println("filters done")
    tempo = aubio.TempoOrDie(aubio.Complex, uint(*Bufsize), uint(*Blocksize), uint(*Samplerate))
    tempo.SetSilence(-90.0)
    tempo.SetThreshold(0.0)
    return c, p
}

func getFilters() (*aubio.Pitch, *aubio.PhaseVoc, *aubio.FilterBank) {
    pitcher := aubio.NewPitch(
        aubio.PitchDefault,
        uint(*Bufsize),
        uint(*Blocksize),
        uint(*Samplerate),
    )
    pitcher.SetUnit(aubio.PitchOutFreq)
//  pitcher.SetTolerance(0.99)
    phVocer, _ := aubio.NewPhaseVoc(uint(*Bufsize), uint(*Blocksize))
    fber := aubio.NewFilterBank(40, uint(*Bufsize))
    fber.SetMelCoeffsSlaney(uint(*Samplerate))
    return pitcher, phVocer, fber
}

func PushBpm(buffer []float64) {
    audioBuffer := aubio.NewSimpleBufferData(uint(*Bufsize), buffer)
    tempo.Do(audioBuffer)
    audioBuffer.Free()
}

func GetBpm() (float64, float64) {
    return tempo.GetBpm(), tempo.GetConfidence()
}

func GetPitchVal(buffer []float64) float64 {
    audioBuffer := aubio.NewSimpleBufferData(uint(*Bufsize), buffer)
    pitch.Do(audioBuffer)
    pitchVal := pitch.Buffer().Slice()[0] * 2
    audioBuffer.Free()
    return pitchVal
}

func GetMelEnergies(buffer []float64) []float64 {
    audioBuffer := aubio.NewSimpleBufferData(uint(*Bufsize), buffer)
    phVoc.Do(audioBuffer)
    grain := phVoc.Grain()
    fb.Do(grain)
    melEnergies := fb.Buffer().Slice()
    audioBuffer.Free()
    return melEnergies
}

