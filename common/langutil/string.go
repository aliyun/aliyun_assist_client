package langutil

// SafeTruncateStringInBytes safely truncates string content which is using
// multi-byte encoding, e.g. default encoding UTF-8, into that no more than
// specified byte count.
func SafeTruncateStringInBytes(danger string, maxByteCount int) string {
	if maxByteCount <= 0 {
		return ""
	}

	var safeTruncated string
	remainedByteCount := maxByteCount
	for _, runeChar := range danger {
		runeByteString := string(runeChar)
		runeByteCount := len(runeByteString)
		if runeByteCount > remainedByteCount {
			break
		}
		safeTruncated += runeByteString
		remainedByteCount -= runeByteCount
	}
	return safeTruncated
}
