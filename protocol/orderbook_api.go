package protocol

import (
	"math/big"
	"strconv"
	"time"
	"github.com/novaprotocolio/orderbook/orderbook"
	demo "github.com/novaprotocolio/orderbook/common"
)

// remember that API structs to be offered MUST be exported
type OrderbookAPI struct {
	V      int
	Engine *orderbook.Engine
	OutC chan<- interface{}
}

// Version : return version
func (api *OrderbookAPI) Version() (int, error) {
	return api.V, nil
}

func NewOrderbookAPI(v int, orderbookEngine *orderbook.Engine, outC chan<- interface{}) *OrderbookAPI {
	return &OrderbookAPI{
		V:      v,
		Engine: orderbookEngine,
		OutC: outC,
	}
}

func (api *OrderbookAPI) getRecordFromOrder(order *orderbook.Order, ob *orderbook.Orderbook) map[string]string {
	record := make(map[string]string)
	record["timestamp"] = strconv.FormatUint(order.Item.Timestamp, 10)
	record["price"] = order.Item.Price.String()
	record["quantity"] = order.Item.Quantity.String()
	// retrieve the input order_id, by default it is set when retrieving from orderbook
	record["order_id"] = new(big.Int).SetBytes(order.Key).String()
	record["trade_id"] = order.Item.TradeID
	return record
}

func (api *OrderbookAPI) GetBestAskList(pairName string) []map[string]string {
	ob, _ := api.Engine.GetOrderbook(pairName)
	if ob == nil {
		return nil
	}
	orderList := ob.Asks.MaxPriceList()
	if orderList == nil {
		return nil
	}

	// t.Logf("Best ask List : %s", orderList.String(0))
	cursor := orderList.Head()
	// we have length
	results := make([]map[string]string, orderList.Item.Length)
	for cursor != nil {
		record := api.getRecordFromOrder(cursor, ob)
		results = append(results, record)
		cursor = cursor.GetNextOrder(orderList)
	}
	return results
}

func (api *OrderbookAPI) GetBestBidList(pairName string) []map[string]string {
	ob, _ := api.Engine.GetOrderbook(pairName)
	if ob == nil {
		return nil
	}
	orderList := ob.Bids.MinPriceList()
	// t.Logf("Best ask List : %s", orderList.String(0))
	if orderList == nil {
		return nil
	}
	cursor := orderList.Tail()
	// we have length
	results := make([]map[string]string, orderList.Item.Length)
	for cursor != nil {
		record := api.getRecordFromOrder(cursor, ob)
		results = append(results, record)
		cursor = cursor.GetPrevOrder(orderList)
	}
	return results

}

func (api *OrderbookAPI) GetOrder(pairName, orderID string) map[string]string {
	var result map[string]string
	ob, _ := api.Engine.GetOrderbook(pairName)
	if ob == nil {
		return nil
	}
	key := orderbook.GetKeyFromString(orderID)
	order := ob.GetOrder(key)
	if order != nil {
		result = api.getRecordFromOrder(order, ob)
	}
	return result
}

func (api *OrderbookAPI) sendMessage(msg interface{}) {
	api.OutC <- msg
}

func (api *OrderbookAPI) ProcessOrder(payload map[string]string) map[string]string {
	// add order at this current node first
	// get timestamp in milliseconds
	if payload["timestamp"] == "" {
		payload["timestamp"] = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	}
	msg, err := NewOrderbookMsg(payload)
	if err == nil {
		// try to store into model, if success then process at local and broad cast
		trades, orderInBook := api.Engine.ProcessOrder(payload)
		demo.LogInfo("Orderbook result", "Trade", trades, "OrderInBook", orderInBook)
		
		// broad cast message
		go api.sendMessage(msg)

		return orderInBook
	}

	return nil
}

func (api *OrderbookAPI) CancelOrder(payload map[string]string) error {
	// add order at this current node first
	// get timestamp in milliseconds
	payload["timestamp"] = strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	msg, err := NewOrderbookCancelMsg(payload)
	if err == nil {
		// try to store into model, if success then process at local and broad cast
		err := api.Engine.CancelOrder(payload)
		demo.LogInfo("Orderbook cancel result", "err", err, "msg", msg)

		// broad cast message
		go api.sendMessage(msg)

	}

	return err
}
