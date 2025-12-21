FROM golang:1.23 as builder

RUN apt-get update && apt-get install -y \
    libx11-dev \
    libxrandr-dev \
    libxinerama-dev \
    libxcursor-dev \
    libxi-dev \
    libxxf86vm-dev \
    mesa-common-dev \
    libgl1-mesa-dev \
    libayatana-appindicator3-dev \
    pkg-config \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY . .

ENV APP_NAME=picker

ENV TARGETS="linux/amd64"

RUN mkdir -p /output && \
    for target in $TARGETS; do \
        GOOS=$(echo $target | cut -d'/' -f1); \
        GOARCH=$(echo $target | cut -d'/' -f2); \
        echo "Building for $GOOS/$GOARCH..."; \
        env GOOS=$GOOS GOARCH=$GOARCH go build -o /output/${APP_NAME}-$GOOS-$GOARCH . || exit 1; \
    done
