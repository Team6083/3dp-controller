# syntax=docker/dockerfile:1.7-labs

FROM golang:latest as builder

ENV CGO_ENABLED=0

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

# Build openapi docs
RUN swag init

# Build the application
RUN go build -o main .

FROM openapitools/openapi-generator-cli as openapi_gen

# Move to working directory /build
WORKDIR /build

# Copy docs from builder
COPY --from=builder /build/docs/swagger.yaml .

# Generate api
RUN docker-entrypoint.sh generate -i swagger.yaml -o ./api -g typescript-axios --skip-validate-spec

FROM node:latest as app_builder

# Move to working directory /build
WORKDIR /build

# Copy and install deps
COPY frontend/package.json .
COPY frontend/package-lock.json .
RUN npm install

# Copy the code into the container
COPY frontend .

# Copy api files
COPY --from=openapi_gen /build/api ./src/api

# Build frontend
RUN npm run build

FROM alpine

WORKDIR /dist
COPY --from=builder /build/main .
COPY --from=app_builder /build/dist frontend/dist

# Timezone file
COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /opt/zoneinfo.zip
ENV ZONEINFO /opt/zoneinfo.zip

# Export necessary port
EXPOSE 8080

# Command to run when starting the container
CMD ["/dist/main"]
