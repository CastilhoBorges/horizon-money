# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/horizon-money ./cmd/horizon-money

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/horizon-money /usr/local/bin/horizon-money
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --retries=5 CMD ["horizon-money", "healthcheck"]
ENTRYPOINT ["horizon-money"]
CMD ["serve"]
