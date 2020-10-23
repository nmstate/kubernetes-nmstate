ARG  BRANCH=master
FROM quay.io/nmstate/kubernetes-nmstate-docs:${BRANCH}
COPY Makefile Gemfile* /docs/
RUN make -C docs install
COPY . /docs/
RUN make -C docs check
