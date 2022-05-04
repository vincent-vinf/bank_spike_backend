package main

import (
	"bank_spike_backend/internal/access"
	"bank_spike_backend/internal/db"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"net"
)

var (
	rpcPort int
)

func init() {
	flag.IntVar(&rpcPort, "rpc-port", 8082, "")
	flag.Parse()
}

func main() {
	defer db.Close()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", rpcPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	access.RegisterAccessServer(grpcServer, &accessServer{})
	err = grpcServer.Serve(lis)
	if err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type accessServer struct {
}

func (s *accessServer) IsAccessible(ctx context.Context, req *access.AccessReq) (*access.AccessRes, error) {
	// TODO check accessible
	log.Println(req.UserId, req.SpikeId)
	return &access.AccessRes{Result: true}, nil
}
