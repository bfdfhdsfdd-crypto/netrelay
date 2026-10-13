FROM golang:1.26.5-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /netrelay .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /netrelay /netrelay
USER nonroot:nonroot
EXPOSE 1111
ENTRYPOINT ["/netrelay"]
