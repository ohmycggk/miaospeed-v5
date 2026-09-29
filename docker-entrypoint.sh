#!/bin/sh
set -e

# Shorthand: `docker run <image> -token xxx` == `miaospeed server -token xxx`.
if [ "${1#-}" != "$1" ]; then
    set -- server "$@"
fi

# True when an argv element equals the flag. Values are not scanned, so a
# token that happens to contain the flag text does not count as present.
flag_present() {
    _flag=$1
    shift
    for _arg in "$@"; do
        if [ "$_arg" = "$_flag" ]; then
            return 0
        fi
    done
    return 1
}

is_truthy() {
    case "$1" in
        1|true|TRUE|True|yes|YES|Yes|on|ON|On) return 0 ;;
        *) return 1 ;;
    esac
}

if [ "$1" = "server" ]; then
    # Env-driven defaults; an explicit flag always wins.
    if [ -n "$MIAOSPEED_META" ]; then
        echo "miaospeed: MIAOSPEED_META is set; this image already uses mihomo and will not switch binaries." >&2
    fi

    if ! flag_present -mtls "$@"; then
        if is_truthy "$MIAOSPEED_MTLS"; then
            set -- "$@" -mtls
        fi
    fi
    if ! flag_present -nospeed "$@"; then
        if is_truthy "$MIAOSPEED_NOSPEED"; then
            set -- "$@" -nospeed
        fi
    fi
    if ! flag_present -verbose "$@"; then
        if is_truthy "$MIAOSPEED_VERBOSE"; then
            set -- "$@" -verbose
        fi
    fi

    if ! flag_present -token "$@"; then
        _token=${MIAOSPEED_TOKEN:-$TOKEN}
        if [ -n "$_token" ]; then
            set -- "$@" -token "$_token"
        fi
    fi
    if ! flag_present -connthread "$@"; then
        if [ -n "$MIAOSPEED_CONNTHREAD" ]; then
            set -- "$@" -connthread "$MIAOSPEED_CONNTHREAD"
        fi
    fi
    if ! flag_present -pausesecond "$@"; then
        if [ -n "$MIAOSPEED_PAUSESECOND" ]; then
            set -- "$@" -pausesecond "$MIAOSPEED_PAUSESECOND"
        fi
    fi
    if ! flag_present -whitelist "$@"; then
        if [ -n "$MIAOSPEED_WHITELIST" ]; then
            set -- "$@" -whitelist "$MIAOSPEED_WHITELIST"
        fi
    fi
    if ! flag_present -path "$@"; then
        if [ -n "$MIAOSPEED_PATH" ]; then
            set -- "$@" -path "$MIAOSPEED_PATH"
        fi
    fi
    if ! flag_present -speedlimit "$@"; then
        if [ -n "$MIAOSPEED_SPEEDLIMIT" ]; then
            # moshaoli documents this variable in MB/s. The flag itself is bytes/s.
            set -- "$@" -speedlimit "$((MIAOSPEED_SPEEDLIMIT * 1024 * 1024))"
        fi
    fi
    if ! flag_present -mmdb "$@"; then
        case "$MIAOSPEED_MMDB" in
            "") ;;
            */*|*.mmdb*)
                set -- "$@" -mmdb "$MIAOSPEED_MMDB"
                ;;
            *)
                echo "miaospeed: MIAOSPEED_MMDB=$MIAOSPEED_MMDB is not a database path; ignoring. Mount an mmdb file and set MIAOSPEED_MMDB to its path." >&2
                ;;
        esac
    fi
    if ! flag_present -serverpublickey "$@"; then
        if [ -n "$MIAOSPEED_SERVERPUBLICKEY" ]; then
            set -- "$@" -serverpublickey "$MIAOSPEED_SERVERPUBLICKEY"
        fi
    fi
    if ! flag_present -serverprivatekey "$@"; then
        if [ -n "$MIAOSPEED_SERVERPRIVATEKEY" ]; then
            set -- "$@" -serverprivatekey "$MIAOSPEED_SERVERPRIVATEKEY"
        fi
    fi

    if ! flag_present -bind "$@"; then
        _bind=${MIAOSPEED_BIND:-$BIND}
        if [ -z "$_bind" ]; then
            _bind=0.0.0.0:9966
        fi
        set -- "$@" -bind "$_bind"
    fi
fi

# Route miaospeed subcommands (server/script/misc/...) through the binary;
# anything that is a real program (e.g. `sh`) is executed as-is.
if ! command -v "$1" >/dev/null 2>&1; then
    set -- miaospeed "$@"
fi

exec "$@"
