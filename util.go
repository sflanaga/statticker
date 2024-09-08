package statticker

import "fmt"

func FormatBytes[T int64 | uint64](bytes T) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2fTB", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.2fGB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2fMB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2fKB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

func AddCommas[T int | uint | int32 | uint32 | int64 | uint64](i T) string {
	src := fmt.Sprint(i)
	n := len(src)
	if n == 0 {
		return ""
	}

	// Calculate size of the destination slice: original length + commas
	numCommas := (n - 1) / 3
	dst := make([]byte, n+numCommas)

	// Initialize position for the last character in the destination slice
	dstPos := len(dst) - 1
	// Start copying characters from the end of the source string
	for i := n - 1; i >= 0; i-- {
		// Copy character
		dst[dstPos] = src[i]
		dstPos--
		// Insert comma if necessary
		if (n-i)%3 == 0 && i > 0 {
			dst[dstPos] = ','
			dstPos--
		}
	}

	// Convert byte slice to string and return
	return string(dst[dstPos+1:])
}
