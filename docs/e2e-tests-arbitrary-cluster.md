```bash
# Configure kubeconfig
export KUBECONFIG=~/.kube/config
export KUBECTL=$(which kubectl)
export SSH=hack/ssh.sh

# Get available NICs
nics=($($KUBECTL get nns -o=jsonpath='{.items[0].status.currentState.interfaces[?(@.type=="ethernet")].name}'))
export PRIMARY_NIC="${nics[0]}"
export FIRST_SECONDARY_NIC="${nics[1]}"
export SECOND_SECONDARY_NIC="${nics[2]}"

# Run tests
FOCUS_1='-ginkgo.focus Nodes.*when.*are.*up.*and.*new.*interface.*is.*configured.*should.*update.*node.*network.*state.*with.*it'
FOCUS_2='-ginkgo.focus rollback.*when.*connectivity.*to.*default.*gw.*is.*lost.*after.*state.*configuration.*should.*rollback.*to.*a.*good.*gw.*configuration'
FOCUS_3='-ginkgo.focus NodeSelector.*when.*policy.*is.*set.*with.*node.*selector.*not.*matching.*any.*nodes.*should.*not.*update.*any.*nodes.*and.*have.*false.*Matching.*state'
FOCUS_4='-ginkgo.focus EnactmentCondition.*when.*applying.*valid.*config.*should.*go.*from.*Progressing.*to.*Available'

make test/e2e E2E_TEST_TIMEOUT=60m E2E_TEST_ARGS="$FOCUS_3" NAMESPACE=my-custom-namespace KUBECONFIG=$KUBECONFIG
```
