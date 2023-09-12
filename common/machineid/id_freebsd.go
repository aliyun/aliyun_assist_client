package machineid


const (
	hostidPath = "/etc/hostid"
)

// machineID returns the hostid specified at `/etc/hostid`.
// If there is an error reading the files an empty string is returned.
func machineID() (string, error) {
	id, err := readFile(hostidPath)
	if err != nil {
		return "", err
	}
	return trim(string(id)), nil
}