package protocol

import (
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/novaprotocolio/orderbook/orderbook"
)

// the service we want to offer on the node
// it must implement the node.Service interface
type OrderbookService struct {
	V      int
	Engine *orderbook.Engine
	protos []p2p.Protocol
	OutC chan<- interface{}
}

// APIs : api service
// specify API structs that carry the methods we want to use
func (service *OrderbookService) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "orderbook",
			Version:   "0.1",
			Service:   NewOrderbookAPI(service.V, service.Engine, service.OutC),
			Public:    true,
		},
	}
}

// these are needed to satisfy the node.Service interface
// in this example they do nothing
func (service *OrderbookService) Protocols() []p2p.Protocol {
	return service.protos
}

func (service *OrderbookService) Start(srv *p2p.Server) error {
	return nil
}

func (service *OrderbookService) Stop() error {
	return nil
}

// NewService: wrapper function for servicenode to start the service, both APIs and Protocols
func NewService(quitC <-chan struct{}, orderbookEngine *orderbook.Engine) func(ctx *node.ServiceContext) (node.Service, error) {
	
	msgC := make(chan interface{})

	proto := NewProtocol(msgC, quitC, orderbookEngine)
	var protocolArr []p2p.Protocol
	if proto != nil {
		protocolArr = []p2p.Protocol{*proto}
	}

	return func(ctx *node.ServiceContext) (node.Service, error) {
		return &OrderbookService{
			V:      1,
			Engine: orderbookEngine,
			protos: protocolArr,
			OutC: msgC,
		}, nil
	}
}
