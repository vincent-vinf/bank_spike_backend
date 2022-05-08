package main

import (
	"bank_spike_backend/cmd/access/filter"
	"bank_spike_backend/internal/access"
	"bank_spike_backend/internal/db"
	redisx "bank_spike_backend/internal/redis"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/patrickmn/go-cache"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

var (
	rpcPort int

	// hold 活动对应的Filter，防止反复生成浪费算力
	// map存储无法过期，随着活动数量增多，内存会大量浪费
	// filterMap = make(map[string]filter.Filter)
	filterCache = cache.New(5*time.Minute, 10*time.Minute)
)

func init() {
	flag.IntVar(&rpcPort, "rpc-port", 8082, "")
	flag.Parse()
}

func main() {
	defer db.Close()
	defer redisx.Close()
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

type accessServer struct{}

func (s *accessServer) IsAccessible(ctx context.Context, req *access.AccessReq) (*access.AccessRes, error) {
	log.Printf("check accessible user: %s, spike: %s\n", req.UserId, req.SpikeId)
	f, isFound := filterCache.Get(req.SpikeId)
	var ft filter.Filter
	if isFound {
		ft = f.(filter.Filter)
	} else {
		var err error
		ft, err = NewFilterChain(req.SpikeId)
		if err != nil {
			return nil, err
		}
		filterCache.Set(req.SpikeId, ft, cache.DefaultExpiration)
	}
	u, err := db.GetUserById(req.UserId)
	if err != nil {
		return nil, err
	}
	res, reason, err := ft.Execute(ctx, u)
	if err != nil {
		return nil, err
	}
	return &access.AccessRes{Result: res, Reason: reason}, nil
}

// [{"name":"base", "rule":"{\"age\":{\"min\":0, \"max\":20}, \"workStatus\":{\"not\":[\"无业\"]}}"}]

func NewFilterChain(spikeId string) (head filter.Filter, err error) {
	spike, err := db.GetSpikeById(spikeId)
	if err != nil {
		return nil, err
	}
	var infoList []filter.Info
	err = json.Unmarshal([]byte(spike.AccessRule), &infoList)
	if err != nil {
		return nil, err
	}
	if len(infoList) == 0 || infoList[0].Name != "base" {
		return nil, errors.New("access rule error, spike_id: " + spikeId)
	}
	if f, ok := filter.Map["base"]; ok {
		log.Println("rule: " + infoList[0].Rule)
		head, err = f(infoList[0].Rule)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("base filter not found")
	}
	prv := head
	for i := 1; i < len(infoList); i++ {
		if f, ok := filter.Map[infoList[i].Name]; ok {
			ft, err := f(infoList[i].Rule)
			if err != nil {
				return nil, err
			}
			prv.SetNext(ft)
			prv = ft
		} else {
			return nil, errors.New("filter not found, name: " + infoList[i].Name)
		}
	}
	return
}
