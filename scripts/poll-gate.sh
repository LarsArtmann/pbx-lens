#!/usr/bin/env bash
# poll-gate.sh — wait for the load policy, then EXEC the full gate.
#
# Why a wrapper instead of a shell loop: the receipt rule says the START LOAD a
# gate run records must be the load the loop actually verified. A plain
# "while load high; sleep; done && buildflow" leaves a poll→launch gap in which
# the machine can get busy again (two sessions paid exactly that gap). exec
# removes the gap by construction: the verified poll and the gate are the same
# process.
#
# Policy (docs/buildflow-dev-gate.md): a FULL run needs the 15-minute load
# average below 40. This script reads that figure, prints the triple it
# verified, and execs the gate only when the policy holds.
set -euo pipefail

threshold=${POLL_GATE_THRESHOLD:-40}
interval=${POLL_GATE_INTERVAL:-30}
max_wait=${POLL_GATE_MAX_WAIT:-0}
load_file=${POLL_GATE_LOAD_FILE:-/proc/loadavg}
gate_exec=${POLL_GATE_EXEC:-"buildflow --build-mode full"}

# Exit codes: 1 = cannot read the load, 2 = usage error, 75 = not ready
# (EX_TEMPFAIL: "try again later" is exactly what a busy machine means).
readonly exit_usage=2
readonly exit_load_unreadable=1
readonly exit_not_ready=75

usage() {
	cat <<'USAGE'
Usage: scripts/poll-gate.sh [--threshold N] [--interval SECONDS]
                            [--max-wait SECONDS] [--once] [--self-test] [-- <gate args>]

Waits until the 15-minute load average drops below the threshold, then execs
the full gate in the same process (no poll→launch gap).

  --threshold N     load ceiling, exclusive (default 40)
  --interval S      poll interval in seconds (default 30)
  --max-wait S      give up after S seconds; 0 waits forever (default 0)
  --once            one check: exit 0 when ready, 75 when not, no gate run
  --self-test       prove the script's own contract with stub load files
  --help            this text

Environment: POLL_GATE_THRESHOLD, POLL_GATE_INTERVAL, POLL_GATE_MAX_WAIT,
POLL_GATE_LOAD_FILE, POLL_GATE_EXEC (default "buildflow --build-mode full").
Arguments after -- are passed to the gate command unchanged.
USAGE
}

die_usage() {
	echo "poll-gate: $*" >&2
	usage >&2
	exit "$exit_usage"
}

# read_load prints "1m 5m 15m" from the load file; a broken source is fatal
# because a gate armed on an unread load would defeat the whole point.
read_load() {
	local line
	if ! line=$(head -n 1 "$load_file" 2>/dev/null); then
		echo "poll-gate: cannot read $load_file; refusing to arm the gate blind" >&2
		exit "$exit_load_unreadable"
	fi

	set -- $line
	if [ "$#" -lt 3 ]; then
		echo "poll-gate: $load_file does not hold three load figures" >&2
		exit "$exit_load_unreadable"
	fi

	printf '%s %s %s\n' "$1" "$2" "$3"
}

# ready reports whether the 15-minute figure is below the threshold.
ready() {
	awk -v figure="$1" -v ceiling="$threshold" 'BEGIN { exit !(figure < ceiling) }'
}

self_test() {
	local dir failures=0
	dir=$(mktemp -d)
	trap 'rm -rf "${dir:-}"' EXIT

	# Not ready: over-policy load must refuse the gate, never run it.
	printf '41.00 40.50 40.20 1/100 1\n' >"$dir/busy"
	if POLL_GATE_LOAD_FILE="$dir/busy" POLL_GATE_EXEC="echo GATE-RAN" \
		"$0" --once >/dev/null 2>&1; then
		echo "FAIL: over-policy load armed the gate" >&2
		failures=$((failures + 1))
	else
		local status=$?
		if [ "$status" -ne "$exit_not_ready" ]; then
			echo "FAIL: over-policy load exited $status, want $exit_not_ready" >&2
			failures=$((failures + 1))
		else
			echo "PASS: over-policy load refuses with $exit_not_ready"
		fi
	fi

	# Ready: the gate runs, and the receipt line names the verified load.
	printf '1.00 2.00 3.00 1/100 1\n' >"$dir/quiet"
	local out
	if ! out=$(POLL_GATE_LOAD_FILE="$dir/quiet" POLL_GATE_EXEC="echo" "$0" --once); then
		echo "FAIL: quiet machine did not arm the gate" >&2
		failures=$((failures + 1))
	else
		case "$out" in
		*"START LOAD: 1m=1.00 5m=2.00 15m=3.00"*) echo "PASS: ready machine prints the verified load" ;;
		*)
			echo "FAIL: ready machine output: $out" >&2
			failures=$((failures + 1))
			;;
		esac
	fi

	# Unreadable source: fatal, never silently "ready".
	if POLL_GATE_LOAD_FILE="$dir/missing" "$0" --once >/dev/null 2>&1; then
		echo "FAIL: a missing load source armed the gate" >&2
		failures=$((failures + 1))
	else
		local status=$?
		if [ "$status" -ne "$exit_load_unreadable" ]; then
			echo "FAIL: missing source exited $status, want $exit_load_unreadable" >&2
			failures=$((failures + 1))
		else
			echo "PASS: missing load source refuses with $exit_load_unreadable"
		fi
	fi

	if [ "$failures" -ne 0 ]; then
		echo "poll-gate self-test: $failures failure(s)" >&2
		exit 1
	fi

	echo "poll-gate self-test: all checks passed"
}

once=0
gate_args=()

while [ "$#" -gt 0 ]; do
	case "$1" in
	--threshold)
		[ "$#" -ge 2 ] || die_usage "--threshold needs a value"
		threshold=$2
		shift 2
		;;
	--interval)
		[ "$#" -ge 2 ] || die_usage "--interval needs a value"
		interval=$2
		shift 2
		;;
	--max-wait)
		[ "$#" -ge 2 ] || die_usage "--max-wait needs a value"
		max_wait=$2
		shift 2
		;;
	--once)
		once=1
		shift
		;;
	--self-test)
		self_test
		exit 0
		;;
	--help | -h)
		usage
		exit 0
		;;
	--)
		shift
		gate_args=("$@")
		break
		;;
	*)
		die_usage "unknown argument: $1"
		;;
	esac
done

started=$(date +%s)

while :; do
	load=$(read_load)

	if ready "$load"; then
		read -r l1 l5 l15 <<<"$load"
		echo "poll-gate: START LOAD: 1m=$l1 5m=$l5 15m=$l15 (policy: 15m < $threshold)"

		if [ "$once" -eq 1 ]; then
			echo "poll-gate: ready (single check, gate not run)"
			exit 0
		fi

		echo "poll-gate: exec $gate_exec ${gate_args[*]-}"
		read -r -a gate <<<"$gate_exec"
		exec "${gate[@]}" "${gate_args[@]}"
	fi

	read -r _ _ l15 <<<"$load"
	echo "poll-gate: load 15m=$l15 >= $threshold; waiting ${interval}s" >&2

	if [ "$once" -eq 1 ]; then
		exit "$exit_not_ready"
	fi

	if [ "$max_wait" -gt 0 ] && [ $(($(date +%s) - started)) -ge "$max_wait" ]; then
		echo "poll-gate: gave up after ${max_wait}s; the machine never reached the policy" >&2
		exit "$exit_not_ready"
	fi

	sleep "$interval"
done
