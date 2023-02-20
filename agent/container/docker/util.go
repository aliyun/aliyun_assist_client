package docker

import (
	"strings"
)

// Container names in output of `docker ps`, `docker container list` or
// `docker list` are not prefixed by a slash '/'. HOWEVER container names got
// from docker's client.ContainerList() does have it, i.e., what you see is NOT
// what you get. Looking inside the source of docker CLI, magics are applied to
// these names before presented to Muggles, and that's what we need in our
// humble post-processing phrase on the response of docker API.
// See link below for detail implementation in docker CLI:
// https://github.com/docker/cli/blob/67cc8b1fd88aea06690eaf3e5d56acd68a0178d2/cli/command/formatter/container.go#L125-L148
func StripAndSelectName(names []string) string {
	for _, name := range names {
		prefixTrimmedName := name[1:]
		if (len(strings.Split(prefixTrimmedName, "/")) == 1) {
			return prefixTrimmedName
		}
	}

	// No, there should be at least one container name without slash in the
	// middle of it.
	return names[0][1:]
}
