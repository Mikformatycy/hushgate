FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /gate ./cmd/gate

FROM gcr.io/distroless/static
COPY --from=build /gate /gate
COPY policy.json /etc/gate/policy.json
EXPOSE 8080
ENTRYPOINT ["/gate"]
