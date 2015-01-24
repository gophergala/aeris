package format

type Format struct {
	Container string
	Audio     AudioFormat
	Video     VideoFormat
}

type VideoFormat struct {
	Encoding   string
	Resolution string
}

type AudioFormat struct {
	Encoding string
	Bitrate  int
}
