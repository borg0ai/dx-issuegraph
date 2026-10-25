#!/usr/bin/env bash
# Shared hermetic environment setup for broad local/CI test wrappers.

if [[ -n "${ISSUEGRAPH_CI_TEST_ENV_SH_LOADED:-}" ]]; then
    return 0
fi
ISSUEGRAPH_CI_TEST_ENV_SH_LOADED=1

issuegraph_test_env_enter() {
    if [[ "${ISSUEGRAPH_TEST_ENV_DISABLE:-0}" == "1" ]]; then
        return 0
    fi
    if [[ "${ISSUEGRAPH_TEST_ENV_ACTIVE:-0}" == "1" ]]; then
        return 0
    fi

    local root
    root="$(mktemp -d "${TMPDIR:-/tmp}/issuegraph-test-env-XXXXXX")"
    export ISSUEGRAPH_TEST_ENV_ROOT="$root"
    export ISSUEGRAPH_TEST_ENV_ACTIVE=1

    if [[ -z "${GOCACHE:-}" ]]; then
        local go_cache
        go_cache="$(go env GOCACHE 2>/dev/null || true)"
        if [[ -n "$go_cache" ]]; then
            export GOCACHE="$go_cache"
        fi
    fi
    if [[ -z "${GOMODCACHE:-}" ]]; then
        local go_mod_cache
        go_mod_cache="$(go env GOMODCACHE 2>/dev/null || true)"
        if [[ -n "$go_mod_cache" ]]; then
            export GOMODCACHE="$go_mod_cache"
        fi
    fi

    mkdir -p "$root/home" "$root/xdg-config" "$root/dolt-root"
    : >"$root/gitconfig"

    export HOME="$root/home"
    export USERPROFILE="$root/home"
    export XDG_CONFIG_HOME="$root/xdg-config"
    export DOLT_ROOT_PATH="$root/dolt-root"
    export GIT_CONFIG_NOSYSTEM=1
    export GIT_CONFIG_GLOBAL="$root/gitconfig"
    export ISSUEGRAPH_TEST_IGNORE_REPO_CONFIG=1
    if [[ "${ISSUEGRAPH_TEST_ENV_RUN_DOLT:-0}" != "1" ]]; then
        issuegraph_test_env_add_skip "dolt"
    fi

    unset ISSUEGRAPH_DIR
    unset ISSUEGRAPH_DB
    unset ISSUEGRAPH_DB
    unset ISSUEGRAPH_JSON
    unset ISSUEGRAPH_NO_DB
    unset ISSUEGRAPH_NO_DAEMON
    unset ISSUEGRAPH_ACTOR
    unset ISSUEGRAPH_ACTOR
    unset GT_ROOT
    unset ISSUEGRAPH_DOLT_SHARED_SERVER
    unset ISSUEGRAPH_DOLT_SERVER_MODE
    unset ISSUEGRAPH_DOLT_AUTO_START
    unset ISSUEGRAPH_DOLT_SERVER_HOST
    unset ISSUEGRAPH_DOLT_SERVER_PORT
    unset ISSUEGRAPH_DOLT_PORT
    unset ISSUEGRAPH_DOLT_SERVER_DATABASE
    unset ISSUEGRAPH_DOLT_SERVER_SOCKET
    unset ISSUEGRAPH_DOLT_PASSWORD

    if command -v dolt >/dev/null 2>&1; then
        dolt config --global --add user.name "issuegraph-test" >/dev/null 2>&1 || true
        dolt config --global --add user.email "test@issuegraph.local" >/dev/null 2>&1 || true
    fi

    trap issuegraph_test_env_cleanup EXIT
}

issuegraph_test_env_add_skip() {
    local service="$1"
    local current=",${ISSUEGRAPH_TEST_SKIP:-},"
    if [[ "$current" != *",$service,"* ]]; then
        if [[ -n "${ISSUEGRAPH_TEST_SKIP:-}" ]]; then
            export ISSUEGRAPH_TEST_SKIP="${ISSUEGRAPH_TEST_SKIP},${service}"
        else
            export ISSUEGRAPH_TEST_SKIP="$service"
        fi
    fi
}

issuegraph_test_env_cleanup() {
    if [[ "${ISSUEGRAPH_TEST_ENV_KEEP:-0}" == "1" ]]; then
        return 0
    fi
    if [[ -n "${ISSUEGRAPH_TEST_ENV_ROOT:-}" ]]; then
        rm -rf "$ISSUEGRAPH_TEST_ENV_ROOT"
        unset ISSUEGRAPH_TEST_ENV_ROOT
    fi
}
