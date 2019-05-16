#!/usr/bin/env bash

# Copyright 2019 The nmstate Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

TESTS_IMAGE_NAME=kubernetes-nmstate-functional-tests

function test_image_available {
    docker inspect $TESTS_IMAGE_NAME &> /dev/null
}

function previous_tests_hash {
    cat hack/.tests_hash
}

function current_tests_hash {
    sha1sum tests/* Gopkg.lock | sha1sum
}

function save_tests_hash {
    echo "$(current_tests_hash)" > hack/.tests_hash
}

function previous_test_image_hash {
    cat hack/.test_image_hash
}

function current_test_image_hash {
    docker inspect $TESTS_IMAGE_NAME --format='{{.ID}}' | sha1sum
}

function save_test_image_hash {
    echo "$(current_test_image_hash)" > hack/.test_image_hash
}

# rebuild test image only if tests changed
if [[ ! test_image_available ||
          "$(previous_tests_hash)" != "$(current_tests_hash)" ||
          "$(previous_test_image_hash)" != "$(current_test_image_hash)" ]]; then
    docker build -f tests/Dockerfile -t $TESTS_IMAGE_NAME .
    save_tests_hash
    save_test_image_hash
fi

mkdir -p _out/tests

docker run \
   --network host \
   --mount type=bind,source=$KUBECONFIG,target=/kubeconfig,readonly \
   --mount type=bind,source=$(pwd)/_out/tests,target=/artifacts \
   --mount type=bind,source=$(pwd)/manifests/examples,target=/manifests/examples \
   $TESTS_IMAGE_NAME \
       -kubeconfig=/kubeconfig \
       ${FUNC_TEST_ARGS}
