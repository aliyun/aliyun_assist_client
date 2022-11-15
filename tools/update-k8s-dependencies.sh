#!/bin/sh
set -euo pipefail

VERSION=${1#"v"}
if [ -z "$VERSION" ]; then
    echo "Must specify version!"
    exit 1
fi
MODS=($(
    curl -sS https://raw.githubusercontent.com/kubernetes/kubernetes/v${VERSION}/go.mod |
    sed -n 's|.*k8s.io/\(.*\) => ./staging/src/k8s.io/.*|k8s.io/\1|p'
))
for MOD in "${MODS[@]}"; do
    V=$(
        go mod download -json "${MOD}@kubernetes-${VERSION}" |
        sed -n 's|.*"Version": "\(.*\)".*|\1|p'
    )
    go mod edit "-replace=${MOD}=${MOD}@${V}"
done
go get "k8s.io/kubernetes@v${VERSION}"

# Usage: tools/update-k8s-dependencies.sh <K8S_VERSION>
# In the kubernetes repository they use `v0.0.0` in requrie directives in
# go.mod for staging Go package, which forces API consumers pin the real
# dependencies in their own go.mod file.
# Thanks for [abursavich's suprising comment](https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-521493597)
# solving the problem, which provides what a neat script to handle it.
# See https://github.com/kubernetes/kubernetes/issues/79384 for details
