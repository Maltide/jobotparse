FROM --platform=$BUILDPLATFORM golang:1.25.5-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

RUN set -eux; \
    GOARM=""; \
    if [ "$TARGETARCH" = "arm" ]; then \
      case "$TARGETVARIANT" in \
        v7) GOARM=7 ;; \
        v6) GOARM=6 ;; \
        *) GOARM=7 ;; \
      esac; \
    fi; \
    CGO_ENABLED=0 GOOS="$TARGETOS" GOARCH="$TARGETARCH" ${GOARM:+GOARM=$GOARM} go build -o /out/main .

FROM debian:bookworm-slim
WORKDIR /app

# Runtime-зависимости:
# - poppler-utils => pdftotext (PDF -> text)
# - texlive-* => pdflatex (LaTeX -> PDF), стабильнее на ARMv7 чем tectonic в Alpine
# - ca-certificates => HTTPS-запросы (Ollama/SuperJob)
# - fonts-dejavu => базовые шрифты (в т.ч. кириллица)
RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
    poppler-utils \
    texlive-latex-base \
    texlive-latex-recommended \
    texlive-lang-cyrillic \
    fonts-dejavu-core \
  && rm -rf /var/lib/apt/lists/*

COPY --from=build /out/main ./main
COPY --from=build /app/static ./static

EXPOSE 8080
CMD ["./main"]