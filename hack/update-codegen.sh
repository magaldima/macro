set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

bash -x ${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/magaldima/macro/pkg/client github.com/magaldima/macro/pkg/apis \
  "microservice:v1alpha1" \
  --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt
