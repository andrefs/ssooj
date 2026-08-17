# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS gen
WORKDIR /src
COPY gen/ ./gen/
COPY index.tmpl.html .
WORKDIR /src/gen
RUN go build -o /out/gen .

FROM nginx:alpine
WORKDIR /site
COPY index.tmpl.html .
COPY --from=gen /out/gen ./gen
RUN --mount=type=secret,id=ssooj_env \
    cp /run/secrets/ssooj_env .env && \
    ./gen && \
    cp dist/index.html /usr/share/nginx/html/index.html && \
    rm -f .env gen
