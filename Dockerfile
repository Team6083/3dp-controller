LABEL authors="kennhuang"

FROM golang:latest as builder

# Move to working directory /build
WORKDIR /build

# Copy and download dependency using go mod
COPY go.mod .
COPY go.sum .
RUN go mod download

# Install swag
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy the code into the container
COPY --exclude=frontend . .

# Build the application
RUN go build -o main .

# Build openapi docs
RUN swag init

FROM node:latest as app_builder

# Move to working directory /build
WORKDIR /build

# Copy and install deps
COPY frontend/package.json .
COPY frontend/package-lock.json .
RUN npm install

# Copy the code into the container
COPY frontend .

# Copy openapi docs
COPY --from=builder /build/docs ../docs

# Generate API files
RUN npm run api-gen

# Build frontend
RUN npm run build

FROM alpine

WORKDIR /dist
COPY --from=builder /build/main .
COPY --from=app_builder /build/out frontend/out

# Timezone file
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /opt/zoneinfo.zip
ENV ZONEINFO /opt/zoneinfo.zip

# Export necessary port
EXPOSE 3000

# Command to run when starting the container
CMD ["/dist/main"]