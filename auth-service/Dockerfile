# dispatcher/Dockerfile
FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest

# Güvenlik için sertifikaları ekliyoruz (HTTPS istekleri atılacaksa gerekebilir).
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Builder aşamasında oluşturduğumuz 'main' dosyasını buraya kopyalıyoruz.
COPY --from=builder /app/main .

# Dispatcher 8080 portunda çalışacak demiştik.
EXPOSE 8080

# Uygulamayı başlatıyoruz.
CMD ["./main"]