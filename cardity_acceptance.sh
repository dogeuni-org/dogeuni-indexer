#!/usr/bin/env bash
set -euo pipefail

# Requirements: curl, jq, grep
# Usage:
#   export INDEXER=http://127.0.0.1:8080
#   bash cardity_acceptance.sh
# Notes:
#   - 脚本会在 /tmp/cardity_plans 生成合法/非法 plan 样例，供钱包/SDK 广播
#   - 广播后回填 deploy 的 <contract_id>(=deploy txid) 与 invoke 的 <txid>，脚本将自动做验收

INDEXER="${INDEXER:-}"
if [ -z "${INDEXER}" ]; then
  echo "ERROR: please export INDEXER, e.g. export INDEXER=http://127.0.0.1:8080" >&2
  exit 1
fi

OUT_DIR="/tmp/cardity_plans"
mkdir -p "${OUT_DIR}"

log() { printf "[%s] %s\n" "$(date +%H:%M:%S)" "$*"; }
hr() { printf -- "------------------------------------------------------------\n"; }

gen_plans() {
  hr
  log "Generating sample plans into ${OUT_DIR}"

  # 非法 plan（decode 失败）
  cat > "${OUT_DIR}/illegal_missing_op.json" <<'EOF'
{"p":"cardity"}
EOF
  cat > "${OUT_DIR}/illegal_no_op.json" <<'EOF'
{"p":"cardity","protocol":"Demo","version":"1.0.0"}
EOF

  # 合法 deploy（无 carc，用于最小链路；若有 carc_b64 可自行替换）
  cat > "${OUT_DIR}/deploy.json" <<'EOF'
{
  "p":"cardity",
  "op":"deploy",
  "protocol":"USDTLikeToken",
  "version":"1.0.0",
  "abi_b64":"eyJtZXRob2RzIjpbeyJuYW1lIjoidHJhbnNmZXIiLCJwYXJhbXMiOlt7Im5hbWUiOiJ0byIsInR5cGUiOiJhZGRyZXNzIn0seyJuYW1lIjoiYW1vdW50IiwidHlwZSI6ImludCJ9XX1dfX0=",
  "module":"USDTLikeToken"
}
EOF

  # 合法 invoke（纯 JSON body）
  cat > "${OUT_DIR}/invoke.json" <<'EOF'
{
  "p":"cardity",
  "op":"invoke",
  "contract_id":"<REPLACE_CONTRACT_ID>",
  "module":"USDTLikeToken",
  "method":"transfer",
  "args":["DxxxxToAddress","1000"]
}
EOF

  # 合法 event（占位）
  cat > "${OUT_DIR}/event.json" <<'EOF'
{
  "p":"cardity",
  "op":"event",
  "contract_id":"<REPLACE_CONTRACT_ID>",
  "event_name":"Transfer",
  "params":{"to":"DxxxxToAddress","amount":1000}
}
EOF

  # 非 CRAC 负例（校验失败不落 size/hash）
  cat > "${OUT_DIR}/deploy_bad_carc.json" <<'EOF'
{"p":"cardity","op":"deploy","protocol":"X","version":"1","carc_b64":"AAAA"}
EOF

  log "Plans:"
  ls -1 "${OUT_DIR}"
}

check_metrics_keys() {
  hr
  log "Checking metrics keys existence"
  curl -sf "${INDEXER}/metrics" | grep -E 'cardity_last_block_lag|cardity_decode_fail_rate|cardity_deploy_total|cardity_invoke_total' >/dev/null \
    && log "OK: metrics keys present" || { echo "FAIL: metrics keys missing"; exit 1; }
}

show_metric_delta() {
  local name="$1"
  local before after
  before="$(curl -sf "${INDEXER}/metrics" | awk -v k="${name}" '$1==k {print $2}' | head -n1 || true)"
  log "Snapshot BEFORE ${name}: ${before:-<none>}"
  read -p "Perform action to change ${name} (e.g., broadcast plans). Press ENTER when done." _
  after="$(curl -sf "${INDEXER}/metrics" | awk -v k="${name}" '$1==k {print $2}' | head -n1 || true)"
  log "Snapshot AFTER  ${name}: ${after:-<none>}"
}

check_contract_detail() {
  local cid="$1"
  hr
  log "Checking contract detail for ${cid}"
  curl -sf "${INDEXER}/v4/cardity/contract/${cid}" | jq -e '.creator,.contract_ref' >/dev/null \
    && log "OK: contract has creator/contract_ref" || { echo "WARN: contract fields missing"; }
  curl -sf "${INDEXER}/v4/cardity/abi/${cid}" | jq -e '.abi_json,.abi_hash' >/dev/null \
    && log "OK: abi endpoint returns canonical json/hash" || { echo "WARN: abi not found"; }
}

check_contract_list_filters() {
  hr
  log "Checking contracts list filters (protocol/version)"
  curl -sf -X POST "${INDEXER}/v4/cardity/contracts" \
    -H 'Content-Type: application/json' \
    -d '{"protocol":"USDTLikeToken","version":"1.0.0","limit":5,"offset":0}' | jq -e '.data' >/dev/null \
    && log "OK: contracts list returns data" || { echo "WARN: contracts filter returned empty or error"; }
}

check_invocations() {
  local cid="$1"
  hr
  log "Query invocations (method_fqn, pagination)"
  local first
  first="$(curl -sf "${INDEXER}/v4/cardity/invocations/${cid}?method_fqn=USDTLikeToken.transfer&limit=2" | jq -r '.data[0].id // empty')"
  if [ -n "${first}" ]; then
    log "First page OK, first id=${first}"
    curl -sf "${INDEXER}/v4/cardity/invocations/${cid}?limit=2&cursor_id=${first}" | jq -e '.next_cursor' >/dev/null \
      && log "OK: pagination with next_cursor works" || { echo "WARN: next_cursor missing"; }
  else
    echo "WARN: no invocations found yet"
  fi
}

check_events() {
  local cid="$1"
  hr
  log "Query events for ${cid}"
  curl -sf -X POST "${INDEXER}/v4/cardity/events" \
    -H 'Content-Type: application/json' \
    -d "{\"contract_id\":\"${cid}\",\"event_name\":\"Transfer\",\"limit\":20,\"offset\":0}" | jq -e '.data' >/dev/null \
    && log "OK: events query returns" || { echo "WARN: no events or error"; }
}

rate_limit_smoke() {
  hr
  log "Rate-limit smoke: firing 200 concurrent GETs (expect some 429)"
  set +e
  for i in $(seq 1 200); do
    curl -s "${INDEXER}/v4/cardity/invocations/dummy?limit=1" >/dev/null &
  done
  wait
  set -e
  log "Done. Manually check /metrics or logs for 429 if needed."
}

main() {
  gen_plans
  check_metrics_keys

  hr
  log "STEP A: Trigger decode failure by broadcasting illegal plan(s)"
  log "Use wallet/SDK to broadcast illegal plans from: ${OUT_DIR}/illegal_missing_op.json or illegal_no_op.json"
  show_metric_delta "cardity_errors_total{stage=\"decode\"}"
  show_metric_delta "cardity_decode_fail_rate"

  hr
  log "STEP B: Broadcast deploy plan: ${OUT_DIR}/deploy.json"
  read -p "After broadcast, input <contract_id> (= deploy txid): " CONTRACT_ID
  check_contract_detail "${CONTRACT_ID}"
  check_contract_list_filters
  show_metric_delta "cardity_deploy_total"

  hr
  log "STEP C: Broadcast invoke plan: ${OUT_DIR}/invoke.json (remember to replace <REPLACE_CONTRACT_ID>)"
  read -p "After broadcast, press ENTER to query invocations..." _
  check_invocations "${CONTRACT_ID}"
  show_metric_delta "cardity_invoke_total"

  hr
  log "STEP D: Broadcast event plan: ${OUT_DIR}/event.json (replace <REPLACE_CONTRACT_ID>)"
  read -p "After broadcast, press ENTER to query events..." _
  check_events "${CONTRACT_ID}"

  rate_limit_smoke

  hr
  log "CRAC negative: optionally broadcast ${OUT_DIR}/deploy_bad_carc.json and ensure size/carc_sha256 omitted; validation errors increased."
  log "All checks done."
}

main
