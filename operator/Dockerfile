FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
WORKDIR /
COPY bin/manager .

COPY deploy/crds/nmstate.io_nodenetwork*.yaml /bindata/kubernetes-nmstate/crds/
COPY deploy/handler/namespace.yaml /bindata/kubernetes-nmstate/namespace/
COPY deploy/handler/operator.yaml /bindata/kubernetes-nmstate/handler/handler.yaml
COPY deploy/handler/service_account.yaml /bindata/kubernetes-nmstate/rbac/
COPY deploy/handler/role.yaml /bindata/kubernetes-nmstate/rbac/
COPY deploy/handler/role_binding.yaml /bindata/kubernetes-nmstate/rbac/


USER nonroot:nonroot

ENTRYPOINT ["/manager"]
