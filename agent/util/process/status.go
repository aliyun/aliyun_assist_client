package process

import "fmt"

// StrStatus function converts integer status code into string description
func StrStatus(status int) string {
	switch status {
	case Success:
		return "Success"
	case Timeout:
		return "Timeout"
	case Fail:
		return "Failed"
	default:
		return fmt.Sprintf("InvalidStatusCode: %d", status)
	}
}
