#!/bin/bash -e
grep ^go go.mod |awk '{print $2}'
