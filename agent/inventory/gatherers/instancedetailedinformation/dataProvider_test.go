package instancedetailedinformation

import "fmt"

func MockTestExecutorWithError(command string, args ...string) ([]byte, error) {
	var result []byte
	return result, fmt.Errorf("Random Error")
}

// createMockExecutor creates an executor that returns the given stdout values on subsequent invocations.
// If the number of invocations exceeds the number of outputs provided, the executor will return the last output.
// For example createMockExecutor("a", "b", "c") will return an executor that returns the following values:
// on first call -> "a"
// on second call -> "b"
// on third call -> "c"
// on every call after that -> "c"
func createMockExecutor(stdout ...string) func(string, ...string) ([]byte, error) {
	var index = 0
	return func(string, ...string) ([]byte, error) {
		if index < len(stdout) {
			index += 1
		}
		return []byte(stdout[index-1]), nil
	}
}
