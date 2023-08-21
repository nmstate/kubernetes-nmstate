#!/usr/bin/env python

# Retrieves the version of NMState and NetworkManager (if available) in the
# handler image by following links from the errata tool (link below).
# Requires one CLI arg which is the X.Y version to use. For example
#
# $ ./nmstate-version.py 4.12
# 4.12.27: nmstate-1.4.4-2.el8_6.x86_64 NetworkManager-1:1.36.0-15.el8_6.x86_64
# 4.12.26: nmstate-1.4.4-2.el8_6.x86_64 NetworkManager-1:1.36.0-15.el8_6.x86_64
# 4.12.25: nmstate-1.4.4-1.el8_6.x86_64 NetworkManager-1:1.36.0-14.el8_6.x86_64
# 4.12.22: nmstate-1.4.4-1.el8_6.x86_64 NetworkManager-1:1.36.0-14.el8_6.x86_64
# etc...
#
# Note that not all versions will be present in the output. I believe this is
# because the image is not rebuilt every release, only if there are changes.
# There are also a very few errata that do not specify a full X.Y.Z version
# and as a result those are ignored by this tool too.
#
# Requires requests and requests-kerberos, as well as a valid kerberos ticket
# for authentication to the errata tool.
#
# https://errata.devel.redhat.com/package/show/openshift-kubernetes-nmstate-handler-rhel-8-container

import re
import requests
from requests_kerberos import HTTPKerberosAuth
import sys
import urllib3

check_version = sys.argv[1]
rhel_version = 8
if float(check_version) > 4.13:
    rhel_version = 9

# Some of the URLs used below use custom CA not always present in the default Python bundle.
# Skipping them and disabling warnings so that console output is not polluted.
urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)

base_url = 'https://errata.devel.redhat.com'
start_page = base_url + f'/package/show/openshift-kubernetes-nmstate-handler-rhel-{rhel_version}-container'
r = requests.get(start_page, auth=HTTPKerberosAuth(), verify=False)
page = r.text
version_re = re.compile('title=".* (\d+\.\d+\.\d+) .*".*\n.*\n.*\n.*href="(/release_engineering/show_released_build/\d+)"', flags=re.M)
links = version_re.findall(page)

for link in links:
    version = link[0]
    build = link[1]
    if not version.startswith(check_version):
        continue
    # Find link to brew build
    build_r = requests.get(base_url + build, auth=HTTPKerberosAuth(), headers={'Accept': 'text/html'}, verify=False)
    build_page = build_r.text
    brew_re = re.compile('href="(https://brewweb.engineering.redhat.com/brew/buildinfo\?buildID=\d+)')
    brews = brew_re.findall(build_page)
    # Find link to x86_64.log
    brew_r = requests.get(brews[0], verify=False)
    brew_page = brew_r.text
    log_re = re.compile('"(https://download.devel.redhat.com/brewroot[^\'"]*/x86_64.log)"')
    logs = log_re.findall(brew_page)
    # Grab the NMState and NetworkManager versions from Brew logs
    log_r = requests.get(logs[0], verify=False)
    log_page = log_r.text
    nmstate_re = re.compile('Installing *: *(nmstate-\d[^ ]*)')
    nmstates = nmstate_re.findall(log_page)
    nm_re = re.compile('Installing *: *(NetworkManager-\d[^ ]*)')
    nms = nm_re.findall(log_page)
    if not nms:
        nm_ver = 'NA'
    else:
        nm_ver = nms[0]
    print(f'{version}: {nmstates[0]} {nm_ver}')

