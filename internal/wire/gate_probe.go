package wire

import "fmt"

func probeAlpha() []string {
	labels := make([]string, 0, 24)
	for hour := range 24 {
		if hour%3 == 0 {
			labels = append(labels, fmt.Sprintf("%02d", hour))
		} else {
			labels = append(labels, "")
		}
	}
	return labels
}
