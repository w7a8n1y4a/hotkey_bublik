# Используем официальный образ Go
FROM golang:1.23 as builder

# Устанавливаем зависимости для X11 и системного трея
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

# Указываем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем исходный код в контейнер
COPY . .

# Указываем имя выходного бинарного файла
ENV APP_NAME=myapp

# Список целевых платформ
ENV TARGETS="linux/amd64"

# Сборка для каждой платформы
RUN mkdir -p /output && \
    for target in $TARGETS; do \
        GOOS=$(echo $target | cut -d'/' -f1); \
        GOARCH=$(echo $target | cut -d'/' -f2); \
        echo "Building for $GOOS/$GOARCH..."; \
        env GOOS=$GOOS GOARCH=$GOARCH go build -o /output/${APP_NAME}-$GOOS-$GOARCH . || exit 1; \
    done

# Второй этап: минимальный образ для отправки файлов
FROM curlimages/curl:latest

# Копируем скомпилированные файлы из предыдущего этапа
COPY --from=builder /output /output

# Рабочая директория
WORKDIR /output

# Сценарий отправки файлов
CMD for file in *; do \
      echo "Uploading $file..."; \
      curl -X POST -F "file=@$file" http://example.com/upload; \
    done

