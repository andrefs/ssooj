#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

rm -rf layer
mkdir -p layer/opt/bin layer/opt/lib

CONTAINER_ID=$(docker run -d --entrypoint /bin/sh public.ecr.aws/lambda/provided:al2023 -c "
dnf install -y poppler-utils 2>/dev/null >/dev/null
mkdir -p /layer/opt/bin /layer/opt/lib
cp /usr/bin/pdftotext /layer/opt/bin/
ldd /usr/bin/pdftotext 2>/dev/null | grep '=> /' | awk '{print \$3}' | while read f; do cp -v \"\$f\" /layer/opt/lib/ 2>/dev/null; done
echo DONE
")

sleep 40
docker cp "$CONTAINER_ID":/layer/opt/bin/pdftotext layer/opt/bin/
docker cp "$CONTAINER_ID":/layer/opt/lib/. layer/opt/lib/
docker rm -f "$CONTAINER_ID" 2>/dev/null

echo "Layer ready: $(ls layer/opt/bin/) + $(ls layer/opt/lib/ | wc -l) libs"
