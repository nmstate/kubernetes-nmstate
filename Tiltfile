def env_bool(name, default):
    value = os.getenv(name, '')
    if value == '':
        return default
    value = value.lower()
    return value in ['1', 'true', 'yes', 'on']

def render_template(path, replacements):
    content = str(read_file(path))
    for key, value in replacements.items():
        content = content.replace('{{ .%s }}' % key, value)
    return blob(content)

def render_operator_manifests(values):
    return [
        render_template('deploy/operator/namespace.yaml', values),
        render_template('deploy/operator/service_account.yaml', values),
        render_template('deploy/operator/role.yaml', values),
        render_template('deploy/operator/role_binding.yaml', values),
        render_template('deploy/operator/operator.yaml', values),
    ]

def object_selector(name, kind, namespace=''):
    kind_name = {
        'clusterrole': 'ClusterRole',
        'clusterrolebinding': 'ClusterRoleBinding',
        'crd': 'CustomResourceDefinition',
        'namespace': 'Namespace',
        'nmstate': 'NMState',
        'role': 'Role',
        'rolebinding': 'RoleBinding',
        'serviceaccount': 'ServiceAccount',
    }.get(kind.lower(), kind)

    cluster_scoped_kinds = [
        'ClusterRole',
        'ClusterRoleBinding',
        'CustomResourceDefinition',
        'NMState',
        'Namespace',
    ]

    if kind_name in cluster_scoped_kinds:
        return '%s:%s:default' % (name, kind_name)

    if namespace:
        return '%s:%s:%s' % (name, kind_name, namespace)

    return '%s:%s' % (name, kind_name)

current_context = k8s_context()
if current_context:
    allow_k8s_contexts(current_context)

if current_context and not current_context.startswith('kind-') and not env_bool('TILT_ALLOW_ANY_CONTEXT', False):
    fail('Current context %s is not a kind cluster. Set TILT_ALLOW_ANY_CONTEXT=1 to override.' % current_context)

operator_namespace = os.getenv('TILT_OPERATOR_NAMESPACE', 'nmstate')
handler_namespace = os.getenv('TILT_HANDLER_NAMESPACE', operator_namespace)
monitoring_namespace = os.getenv('TILT_MONITORING_NAMESPACE', 'monitoring')
image_repo = os.getenv('TILT_IMAGE_REPO', 'quay.io/immortal')
operator_image = '%s/kubernetes-nmstate-operator' % image_repo
handler_image = '%s/kubernetes-nmstate-handler' % image_repo
operator_pull_policy = os.getenv('TILT_OPERATOR_PULL_POLICY', 'IfNotPresent')
handler_pull_policy = os.getenv('TILT_HANDLER_PULL_POLICY', 'IfNotPresent')
kube_rbac_proxy_image = os.getenv(
    'TILT_KUBE_RBAC_PROXY_IMAGE',
    'quay.io/openshift/origin-kube-rbac-proxy:4.10.0',
)
enable_handler = env_bool('TILT_ENABLE_HANDLER', False)
handler_node_selector_key = os.getenv('TILT_HANDLER_NODE_SELECTOR_KEY', '')
handler_node_selector_value = os.getenv('TILT_HANDLER_NODE_SELECTOR_VALUE', '')

common_deps = [
    'api',
    'build',
    'cmd',
    'controllers',
    'go.mod',
    'go.sum',
    'pkg',
    'vendor',
]

docker_build(
    handler_image,
    '.',
    dockerfile='build/Dockerfile',
    match_in_env_vars=True,
    only=common_deps,
    ignore=['_kubevirtci', 'build/_output', 'bundle', 'docs', 'test'],
)

docker_build(
    operator_image,
    '.',
    dockerfile='build/Dockerfile.operator',
    only=common_deps + ['deploy'],
    ignore=['_kubevirtci', 'build/_output', 'bundle', 'docs', 'test'],
)

render_values = {
    'HandlerNamespace': handler_namespace,
    'HandlerImage': handler_image,
    'HandlerPullPolicy': handler_pull_policy,
    'MonitoringNamespace': monitoring_namespace,
    'OperatorNamespace': operator_namespace,
    'OperatorImage': operator_image,
    'OperatorPullPolicy': operator_pull_policy,
    'KubeRBACProxyImage': kube_rbac_proxy_image,
}

k8s_yaml(read_file('deploy/crds/nmstate.io_nmstates.yaml'))
for manifest in render_operator_manifests(render_values):
    k8s_yaml(manifest)

k8s_resource(
    new_name='operator-prereqs',
    objects=[
        object_selector('nmstates.nmstate.io', 'crd'),
        object_selector(operator_namespace, 'namespace'),
        object_selector('nmstate-operator', 'serviceaccount', operator_namespace),
        object_selector('nmstate-operator', 'clusterrole'),
        object_selector('nmstate-operator', 'role', operator_namespace),
        object_selector('nmstate-operator', 'clusterrolebinding', operator_namespace),
        object_selector('nmstate-operator', 'rolebinding', operator_namespace),
    ],
)

k8s_resource(
    workload='nmstate-operator',
    resource_deps=['operator-prereqs'],
)

if enable_handler:
    nmstate = decode_yaml(read_file('deploy/examples/nmstate.io_v1_nmstate_cr.yaml'))
    if handler_node_selector_key:
        nmstate.setdefault('spec', {})
        nmstate['spec']['nodeSelector'] = {
            handler_node_selector_key: handler_node_selector_value,
        }
    k8s_yaml(encode_yaml(nmstate))
    k8s_resource(
        new_name='nmstate-instance',
        objects=[object_selector('nmstate', 'nmstate')],
        resource_deps=['nmstate-operator'],
    )

# Manual helpers that mirror the project's existing dev/test entrypoints.
local_resource(
    'unit-tests',
    cmd='make test/unit WHAT=./controllers/... ./pkg/...',
    deps=['api', 'controllers', 'pkg', 'go.mod', 'go.sum', 'vendor'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)

local_resource(
    'generate',
    cmd='make generate',
    deps=['api', 'controllers', 'hack', 'deploy', 'go.mod', 'go.sum'],
    trigger_mode=TRIGGER_MODE_MANUAL,
    auto_init=False,
)
