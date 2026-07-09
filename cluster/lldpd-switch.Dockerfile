# The kubevirt CI DNS only resolves an allowlist of domains: Fedora
# repositories are resolvable there (quay.io + fedoraproject.org), while e.g.
# the Alpine CDN is not, so stick to a Fedora based image.
FROM quay.io/fedora/fedora:latest

RUN dnf install -y lldpd procps-ng && dnf clean all
