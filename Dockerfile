FROM golang:1.17 as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY cmd cmd
COPY internal internal

RUN mkdir -p "bin" && \
    go build -o bin/access-server cmd/access/main.go && \
    go build -o bin/spike-server cmd/spike/main.go && \
    go build -o bin/user-server cmd/user/main.go && \
    go build -o bin/admin-server cmd/admin/main.go

# access
FROM gcr.io/distroless/static as access
WORKDIR /
COPY --from=builder /app/bin/access-server /
USER root

# spike
FROM gcr.io/distroless/static as spike
WORKDIR /
COPY --from=builder /app/bin/spike-server /
USER root

# user
FROM gcr.io/distroless/static as user
WORKDIR /
COPY --from=builder /app/bin/user-server /
USER root

# admin
FROM gcr.io/distroless/static as admin
WORKDIR /
COPY --from=builder /app/bin/admin-server /
USER root