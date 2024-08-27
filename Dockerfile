# syntax=docker/dockerfile:1

FROM golang:1.23 AS build-stage

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/reference/dockerfile/#copy
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o bin ./...

# Deploy the application binary into a lean image
FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/bin/MergeSentinel /MergeSentinel

EXPOSE 8080

USER nonroot:nonroot

# Run
ENTRYPOINT ["/MergeSentinel"]
