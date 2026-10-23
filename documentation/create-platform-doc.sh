#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DATA="$SCRIPT_DIR/data/platform_data.json"
OUT="$SCRIPT_DIR/docs/reference/platform.md"

cat >> "$OUT" <<'EOF'

| k8s Cluster | Artemis Version | Artemis Cloud Operator Version | PF ActiveMQ Operator Version | Usecases |
|---|---|---|---|---|
EOF

jq -r '.clusters[] | [
  .id,
  .artemisVersion,
  .artemisCloudOperatorVersion,
  .pfActivemqOperatorVersion,
  (if (.id | contains("lab")) then "playground for platform team"
   elif (.id | contains("pci")) then "pci(light) workload"
   else "normal workload" end)
] | "| " + join(" | ") + " |"' "$DATA" >> "$OUT"

echo "Generated $OUT"
