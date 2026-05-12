#!/usr/bin/env bash
# Copyright (c) 2021-2022 Dell Inc., or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

# Defaults
CHAIN="RESILIENCY_METRO"
ACTION=""
LOCAL_IPS=()
REMOTE_IPS=()
SSH_USER=""
SSH_KEY=""
PROTOCOL="any"          # "tcp" | "udp" | "any"
PORT=""                 # e.g., 3260
DRYRUN=0
PARALLEL=4              # concurrent SSH sessions
COMMENT='resiliency-testing-delete-me'

usage() {
  cat <<'EOF'
Usage: block-traffic.sh [options]

Required:
  --local-ips "10.0.0.11 10.0.0.12"      Space-separated IPs of LOCAL array nodes to SSH into
  --remote-ips "10.0.1.21 10.0.1.22"     Space-separated IPs of REMOTE array nodes to block/unblock
  --action <block|unblock>               What to do on local nodes

Auth:
  --ssh-user <user>                      SSH username on local nodes
  --ssh-key  </path/to/key>              Optional: path to private key (default uses agent)
  # If using passwords, set SSHPASS env and install sshpass; script auto-detects it.

Behavior:
  --chain <name>                         Chain name (default: RESILIENCY_METRO)
  --protocol <tcp|udp|any>               Narrow rule to protocol (default: any)
  --port <num>                           Narrow rule to destination port (e.g., 3260)
  --dry-run                              Show what would be done, but don't apply
  --parallel <N>                         Number of SSH sessions in parallel (default: 4)

Examples:
  # Block all traffic from remote iSCSI nodes on local array nodes
  ./block-traffic.sh --local-ips "10.0.0.11 10.0.0.12" \
                     --remote-ips "10.0.1.21 10.0.1.22" \
                     --action block --ssh-user admin

  # Unblock only TCP 3260 (iSCSI) if you previously blocked by port
  ./block-traffic.sh --local-ips "10.0.0.11 10.0.0.12" \
                     --remote-ips "10.0.1.21 10.0.1.22" \
                     --action unblock --protocol tcp --port 3260 --ssh-user admin

Notes:
- Idempotent: re-running won't duplicate rules; unblocking removes only the rules created by this script.
- Requires iptables.
EOF
}

# Parse args
while [[ $# -gt 0 ]]; do
  case "$1" in
    --local-ips)   IFS=' ' read -r -a LOCAL_IPS <<< "$2"; shift 2;;
    --remote-ips)  IFS=' ' read -r -a REMOTE_IPS <<< "$2"; shift 2;;
    --action)      ACTION="$2"; shift 2;;
    --ssh-user)    SSH_USER="$2"; shift 2;;
    --ssh-key)     SSH_KEY="$2"; shift 2;;
    --chain)       CHAIN="$2"; shift 2;;
    --protocol)    PROTOCOL="$2"; shift 2;;
    --port)        PORT="$2"; shift 2;;
    --dry-run)     DRYRUN=1; shift 1;;
    --parallel)    PARALLEL="$2"; shift 2;;
    --help|-h)     usage; exit 0;;
    *) echo "Unknown option: $1"; usage; exit 1;;
  esac
done

# Validate
[[ ${#LOCAL_IPS[@]} -gt 0 ]]     || { echo "ERROR: --local-ips required"; exit 2; }
[[ ${#REMOTE_IPS[@]} -gt 0 ]]    || { echo "ERROR: --remote-ips required"; exit 2; }
[[ -n "$ACTION" ]]               || { echo "ERROR: --action block|unblock required"; exit 2; }
[[ -n "$SSH_USER" ]]             || { echo "ERROR: --ssh-user required"; exit 2; }
[[ "$ACTION" =~ ^(block|unblock)$ ]] || { echo "ERROR: --action must be block or unblock"; exit 2; }
[[ "$PROTOCOL" =~ ^(tcp|udp|any)$ ]] || { echo "ERROR: --protocol must be tcp|udp|any"; exit 2; }
if [[ "$PROTOCOL" != "any" && -z "$PORT" ]]; then
  echo "INFO: --protocol set without --port; rules will match protocol across ALL ports."
fi

SSH_BIN="sshpass -e ssh"

SSH_OPTS=(-o StrictHostKeyChecking=no -o ConnectTimeout=10)


# Build remote script that manages rules directly in INPUT
build_remote_input_rules() {
  local action="$1" protocol="$2" port="$3"
  shift 3
  local remote_ips=("$@")
  {
    echo "set -euo pipefail"
    echo "ACTION='$action'"
    echo "PROTO='$protocol'"
    echo "PORT='$port'"
    echo "DRYRUN='$DRYRUN'"
    echo "COMMENT='$COMMENT'"

    cat <<'LIB'
exists_input() { sudo iptables -C INPUT $1 >/dev/null 2>&1; }

# Build the match spec: source + proto/port (ordering chosen for stable -C/-D matching)
match_for_ip() {
  local ip="$1"
  if [[ "$PROTO" == "any" ]]; then
    # exact spec used for -C/-D: source only
    echo "-s ${ip}/32 -j DROP -m comment --comment \"$COMMENT\""
  elif [[ -n "$PORT" ]]; then
    # protocol with destination port
    echo "-s ${ip}/32 -p ${PROTO} --dport ${PORT} -j DROP -m comment --comment \"$COMMENT\""
  else
    # protocol on all ports
    echo "-s ${ip}/32 -p ${PROTO} -j DROP -m comment --comment \"$COMMENT\""
  fi
}
LIB

    for ip in "${remote_ips[@]}"; do
      echo "IP='$ip'"
      echo 'SPEC="$(match_for_ip "$IP")"'
      if [[ "$action" == "block" ]]; then
        cat <<'BLOCK'
# Insert at top (position 1) if not present
if exists_input "$SPEC"; then
  echo "[SKIP] Rule already present for $IP"
else
  if [[ "$DRYRUN" == "1" ]]; then
    echo "[DRYRUN] iptables -I INPUT 1 $SPEC"
  else
    # Insert in INPUT chain at position 1
    sudo iptables -I INPUT 1 $SPEC
  fi
fi
BLOCK
      else # unblock
        cat <<'UNBLOCK'
# Delete by exact spec; -D INPUT expects the same rule spec
if exists_input "$SPEC"; then
  if [[ "$DRYRUN" == "1" ]]; then
    echo "[DRYRUN] iptables -D INPUT $SPEC"
  else
    sudo iptables -D INPUT $SPEC
  fi
else
  echo "[SKIP] No matching rule found for $IP"
fi
UNBLOCK
      fi
    done

    # Show final INPUT chain
    echo 'sudo iptables -L INPUT --line-numbers -n'
  } | sed 's/^ *$//'
}



# Run remote command on a single local node
run_on_node() {
  local node_ip="$1"
  shift
  local script="$*"
  echo "=== ${ACTION^^} on local node ${node_ip} ==="
  if [[ "$DRYRUN" == "1" ]]; then
    echo "[DRYRUN] Would SSH and run:"
    echo "$script"
    return 0
  fi
  $SSH_BIN "${SSH_OPTS[@]}" "${SSH_USER}@${node_ip}" "bash -s" <<< "$script"
}

# Prepare remote script
REMOTE_SCRIPT="$(build_remote_input_rules "$ACTION" "$PROTOCOL" "$PORT" "${REMOTE_IPS[@]}")"

# Concurrency: run across local nodes
# Simple worker queue using background jobs
pids=()
count=0
for node in "${LOCAL_IPS[@]}"; do
  run_on_node "$node" "$REMOTE_SCRIPT" &
  pids+=($!)
  count=$((count+1))
  if (( count % PARALLEL == 0 )); then
    wait "${pids[@]}"
    pids=()
  fi
done
wait "${pids[@]:-}"

echo "Done."
