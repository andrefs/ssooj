#!/usr/bin/env bash
# Builds the poppler-utils Lambda layer from the AL2023 base image.
# Output: infrastructure/layer/opt/ with pdftotext + shared libraries.

set -euo pipefail
cd "$(dirname "$0")"

rm -rf layer/opt
mkdir -p layer/opt

CONTAINER_ID=$(docker run -d --entrypoint /bin/sh public.ecr.aws/lambda/provided:al2023 -c "
dnf install -y poppler-utils 2>/dev/null >/dev/null
mkdir -p /layer/opt/bin /layer/opt/lib
cp /usr/bin/pdftotext /layer/opt/bin/
ldd /usr/bin/pdftotext 2>/dev/null | grep '=> /' | awk '{print \$3}' | while read f; do cp -v "\$f" /layer/opt/lib/ 2>/dev/null; done
echo DONE
")

echo "Waiting for container to finish..."
sleep 30

docker cp "$CONTAINER_ID":/layer/opt/bin/pdftotext layer/opt/bin/ 2>/dev/null || true
docker cp "$CONTAINER_ID":/layer/opt/lib/. layer/opt/lib/ 2>/dev/null || true
docker rm -f "$CONTAINER_ID" 2>/dev/null

echo "Layer files:"
ls -la layer/opt/bin/
ls layer/opt/lib/ | wc -l
echo "Layer build complete."
