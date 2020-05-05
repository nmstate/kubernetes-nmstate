#!/bin/bash -e
script_dir=$(dirname "$(readlink -f "$0")")

version_type=$1
old_version=$(hack/version.sh)
new_version=$(hack/bump-version.sh $version_type)
release_notes=$(mktemp)

end() {
    rm $release_notes
}

trap end EXIT SIGINT SIGTERM SIGSTOP

$RELEASE_NOTES \
    --format go-template:$script_dir/release-notes.tmpl \
    --release-version $new_version \
    --required-author "" \
    --github-org nmstate \
    --github-repo kubernetes-nmstate \
    --start-rev $old_version \
    --end-rev master \
    --output $release_notes

cat << EOF > version/description
$new_version

TODO: Add description here

EOF

cat $release_notes >> version/description

cat << EOF >> version/description

# Installation

First, install kubernetes-nmstate operator:

\`\`\`
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/$new_version/nmstate.io_nmstates_crd.yaml
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/$new_version/namespace.yaml
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/$new_version/service_account.yaml
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/$new_version/role.yaml
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/$new_version/role_binding.yaml
kubectl apply -f https://github.com/nmstate/kubernetes-nmstate/releases/download/$new_version/operator.yaml
\`\`\`

Once that's done, create an \`NMState\` CR, triggering deployment of
kubernetes-nmstate handler:

\`\`\`
cat <<EOF | kubectl create -f -
apiVersion: nmstate.io/v1alpha1
kind: NMState
metadata:
  name: nmstate
EOF
\`\`\`
EOF

${EDITOR:-vi} version/description

git checkout -b release-$new_version
git commit -a -s -m "Release $new_version"
