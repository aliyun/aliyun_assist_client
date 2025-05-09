package pathutil

var (
	executableDir     string

	crossVersionInboundDir     string
	versionedInboundDir     string

	crossVersionOutboundDir     string
	versionedOutboundDir     string
)

func GetExecutableDir() (string, error) {
	if executableDir != "" {
		return executableDir, nil
	}

	return getCurrentVersionDir()
}

func GetCrossVersionInboundDir() (string, error) {
	if crossVersionInboundDir != "" {
		return crossVersionInboundDir, nil
	}

	return GetCrossVersionDir()
}

func getVersionedInboundDir() (string, error) {
	if versionedInboundDir != "" {
		return versionedInboundDir, nil
	}

	return getCurrentVersionDir()
}

func getCrossVersionOutboundDir() (string, error) {
	if crossVersionOutboundDir != "" {
		return crossVersionOutboundDir, nil
	}

	return GetCrossVersionDir()
}

func getVersionedOutboundDir() (string, error) {
	if versionedOutboundDir != "" {
		return versionedOutboundDir, nil
	}

	return getCurrentVersionDir()
}
