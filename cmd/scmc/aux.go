package main

import (
	"fmt"
	"strings"
)

const (
	Kibibyte float64 = 1024
	Mebibyte         = Kibibyte * 1024
	Gibibyte         = Mebibyte * 1024
	Tebibyte         = Gibibyte * 1024
	Pebibyte         = Tebibyte * 1024
	Exbibyte         = Pebibyte * 1024
)

func bytesToSize(b uint64) string {
	var (
		u string  = ""
		v float64 = float64(b)
	)

	switch {
	case v >= Exbibyte:
		u = "EiB"
		v /= Exbibyte
	case v >= Tebibyte:
		u = "TiB"
		v /= Tebibyte
	case v >= Gibibyte:
		u = "GiB"
		v /= Gibibyte
	case v >= Mebibyte:
		u = "MiB"
		v /= Mebibyte
	case v >= Kibibyte:
		u = "KiB"
		v /= Kibibyte
	default:
		return fmt.Sprintf("%d B", int(v))
	}

	return fmt.Sprintf("%.1f %s", v, u)
}

func oxfordJoin(values []string, format string, w string) string {
	var s strings.Builder

	for i := 0; i < len(values); i++ {
		switch {
		case i > 0 && i < len(values)-1:
			s.WriteString(", ")
		case i > 0:
			s.WriteString(", ")
			s.WriteString(w)
			s.WriteString(" ")
		}

		s.WriteString(fmt.Sprintf(format, values[i]))
	}

	return s.String()
}
