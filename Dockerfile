FROM golang:1.23-alpine

WORKDIR /app

# Copie les dépendances
COPY go.mod go.sum ./
RUN go mod download

# Copie le code source
COPY . .

# Build le binaire
RUN go build -o main .

EXPOSE 8080

ENV PORT=8080

# Lance le binaire compilé
CMD ["./main"]