# booking-service/Dockerfile
FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum* ./

RUN go mod download

COPY . .

RUN go test -v ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:latest

# Güvenlik için sertifikaları ekliyoruz (HTTPS istekleri atılacaksa gerekebilir).
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Builder aşamasında oluşturduğumuz 'main' dosyasını buraya kopyalıyoruz.
COPY --from=builder /app/main .

# Booking Service 8083 portunda çalışacak demiştik.
EXPOSE 8083

# Uygulamayı başlatıyoruz.
CMD ["./main"]