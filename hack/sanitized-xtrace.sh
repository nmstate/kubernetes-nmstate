# -----BEGIN SANITIZED XTRACE-----
# List Perl regexes that you want to replace here.
# Make sure to end each line with a semicolon.
# Also note that things have to be escaped for bash as well.
# TODO:
# - Generate this from https://github.com/leaktk/patterns
# - Support setting set +x to run 'trap - DEBUG'
read -r -d '' REPLACES << EOF
s/(?i)((?:secret|auth|passw|token)\\w+[:=]\\s*?[\\'\\"]?).*?([\\'\\"])/\$1 ...REDACTED...\$2/g;
EOF

__sanitize_xtrace() {
    # Disable xtrace since we're handling it here
    [[ "${SHELLOPTS}" =~ "xtrace" ]] && set +x

    [[ -z "$BASH_COMMAND" ]] && return
    [[ "$BASH_COMMAND" == local\ trap_guard_active=* ]] && return
    [[ "$BASH_COMMAND" == trap* ]] && return

    # Provide sanitized version of xtrace
    echo "${PS4:-+ }${BASH_COMMAND}" | perl -pe "${REPLACES}" >&2
}

trap '__sanitize_xtrace' DEBUG
# -----END SANITIZED XTRACE-----
