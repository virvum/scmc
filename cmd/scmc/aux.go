package main

import (
	"fmt"
	"strings"
)

const (
	kibibyte float64 = 1024
	mebibyte         = kibibyte * 1024
	gibibyte         = mebibyte * 1024
	tebibyte         = gibibyte * 1024
	pebibyte         = tebibyte * 1024
	exbibyte         = pebibyte * 1024
)

func bytesToSize(b uint64) string {
	var (
		u string  = ""
		v float64 = float64(b)
	)

	switch {
	case v >= exbibyte:
		u = "EiB"
		v /= exbibyte
	case v >= tebibyte:
		u = "TiB"
		v /= tebibyte
	case v >= gibibyte:
		u = "GiB"
		v /= gibibyte
	case v >= mebibyte:
		u = "MiB"
		v /= mebibyte
	case v >= kibibyte:
		u = "KiB"
		v /= kibibyte
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
