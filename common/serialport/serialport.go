package serialport

const (
	// Limit the maximum length of a single write to the serial port to prevent
	// serial port from broken. If the limit is exceeded, the data will be truncated.
	MaxLenOfSingleWrite = 1100
)
