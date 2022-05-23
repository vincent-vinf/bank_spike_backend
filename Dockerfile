FROM golang:1.17 as builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go env -w  GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

RUN mkdir -p "bin" && \
    go build -o bin/access-service cmd/access/main.go && \
    go build -o bin/spike-service cmd/spike/main.go && \
    go build -o bin/user-service cmd/user/main.go && \
    go build -o bin/admin-service cmd/admin/main.go && \
    go build -o bin/order-service cmd/order/main.go

# gcr.io/distroless/static
# access
FROM debian as access
WORKDIR /
COPY --from=builder /app/bin/access-service /
USER root

# spike
FROM debian as spike
WORKDIR /
COPY --from=builder /app/bin/spike-service /
USER root

# user
FROM debian as user
WORKDIR /
COPY --from=builder /app/bin/user-service /
USER root

# admin
FROM debian as admin
WORKDIR /
COPY --from=builder /app/bin/admin-service /
USER root

# order
FROM debian as order
WORKDIR /
COPY --from=builder /app/bin/order-service /
USER root

