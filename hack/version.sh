#!/bin/bash -e
grep = version/version.go | sed -r 's/.*= \"(.*)"$$/v\1/g'
