package orderbook

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"strconv"
	"strings"
	"time"
)

type OrderbookItem struct {
	Timestamp     uint64
	NextOrderID   uint64
	MaxPricePoint uint64
	Name          string
}
type Orderbook struct {
	db   *BatchDatabase
	Bids *OrderTree
	Asks *OrderTree
	Item *OrderbookItem
	Key  []byte
	slot *big.Int
}

const (
	Ask         = "ask"
	Bid         = "bid"
	Market      = "market"
	Limit       = "limit"
	SlotSegment = common.AddressLength
)

func NewOrderbook(name string, db *BatchDatabase) *Orderbook {
	item := &OrderbookItem{NextOrderID: 0, Name: strings.ToLower(name)}
	key := crypto.Keccak256([]byte(item.Name))
	slot := new(big.Int).SetBytes(key)
	bidsKey := GetSegmentHash(key, 1, SlotSegment)
	asksKey := GetSegmentHash(key, 2, SlotSegment)
	orderBook := &Orderbook{db: db, Item: item, slot: slot, Key: key}
	bids := NewOrderTree(db, bidsKey, orderBook)
	asks := NewOrderTree(db, asksKey, orderBook)
	orderBook.Bids = bids
	orderBook.Asks = asks
	orderBook.UpdateTime()
	return orderBook
}
func (orderBook *Orderbook) SetDebug(debug bool) {
	orderBook.db.Debug = debug
}
func (orderBook *Orderbook) Save() error {
	orderBook.Asks.Save()
	orderBook.Bids.Save()
	return orderBook.db.Put(orderBook.Key, orderBook.Item)
}
func (orderBook *Orderbook) Commit() error {
	return orderBook.db.Commit()
}
func (orderBook *Orderbook) Restore() error {
	orderBook.Asks.Restore()
	orderBook.Bids.Restore()
	val, err := orderBook.db.Get(orderBook.Key, orderBook.Item)
	if err == nil {
		orderBook.Item = val.(*OrderbookItem)
	}
	return err
}
func (orderBook *Orderbook) GetOrderIDFromBook(key []byte) uint64 {
	orderSlot := new(big.Int).SetBytes(key)
	return Sub(orderSlot, orderBook.slot).Uint64()
}
func (orderBook *Orderbook) GetOrderIDFromKey(key []byte) []byte {
	orderSlot := new(big.Int).SetBytes(key)
	return common.BigToHash(Add(orderBook.slot, orderSlot)).Bytes()
}
func (orderBook *Orderbook) GetOrder(key []byte) *Order {
	if orderBook.db.IsEmptyKey(key) {
		return nil
	}
	storedKey := orderBook.GetOrderIDFromKey(key)
	orderItem := &OrderItem{}
	val, err := orderBook.db.Get(storedKey, orderItem)
	if err != nil {
		fmt.Printf("Key not found :%x, %v\n", storedKey, err)
		return nil
	}
	order := &Order{Item: val.(*OrderItem), Key: key}
	return order
}
func (orderBook *Orderbook) String(startDepth int) string {
	tabs := strings.Repeat("\t", startDepth)
	return fmt.Sprintf("%s{\n\t%sName: %s\n\t%sTimestamp: %d\n\t%sNextOrderID: %d\n\t%sBids: %s\n\t%sAsks: %s\n%s}\n", tabs, tabs, orderBook.Item.Name, tabs, orderBook.Item.Timestamp, tabs, orderBook.Item.NextOrderID, tabs, orderBook.Bids.String(startDepth+1), tabs, orderBook.Asks.String(startDepth+1), tabs)
}
func (orderBook *Orderbook) UpdateTime() {
	timestamp := uint64(time.Now().Unix())
	orderBook.Item.Timestamp = timestamp
}
func (orderBook *Orderbook) BestBid() (value *big.Int) {
	return orderBook.Bids.MaxPrice()
}
func (orderBook *Orderbook) BestAsk() (value *big.Int) {
	return orderBook.Asks.MinPrice()
}
func (orderBook *Orderbook) WorstBid() (value *big.Int) {
	return orderBook.Bids.MinPrice()
}
func (orderBook *Orderbook) WorstAsk() (value *big.Int) {
	return orderBook.Asks.MaxPrice()
}
func (orderBook *Orderbook) processMarketOrder(quote map[string]string, verbose bool) []map[string]string {
	var trades []map[string]string
	quantityToTrade := ToBigInt(quote["quantity"])
	side := quote["side"]
	var newTrades []map[string]string
	zero := Zero()
	if side == Bid {
		for quantityToTrade.Cmp(zero) > 0 && orderBook.Asks.NotEmpty() {
			bestPriceAsks := orderBook.Asks.MinPriceList()
			quantityToTrade, newTrades = orderBook.processOrderList(Ask, bestPriceAsks, quantityToTrade, quote, verbose)
			trades = append(trades, newTrades...)
		}
	} else {
		for quantityToTrade.Cmp(zero) > 0 && orderBook.Bids.NotEmpty() {
			bestPriceBids := orderBook.Bids.MaxPriceList()
			quantityToTrade, newTrades = orderBook.processOrderList(Bid, bestPriceBids, quantityToTrade, quote, verbose)
			trades = append(trades, newTrades...)
		}
	}
	return trades
}
func (orderBook *Orderbook) processLimitOrder(quote map[string]string, verbose bool) ([]map[string]string, map[string]string) {
	var trades []map[string]string
	quantityToTrade := ToBigInt(quote["quantity"])
	side := quote["side"]
	price := ToBigInt(quote["price"])
	var newTrades []map[string]string
	var orderInBook map[string]string
	zero := Zero()
	if side == Bid {
		minPrice := orderBook.Asks.MinPrice()
		for quantityToTrade.Cmp(zero) > 0 && orderBook.Asks.NotEmpty() && price.Cmp(minPrice) >= 0 {
			bestPriceAsks := orderBook.Asks.MinPriceList()
			quantityToTrade, newTrades = orderBook.processOrderList(Ask, bestPriceAsks, quantityToTrade, quote, verbose)
			trades = append(trades, newTrades...)
			minPrice = orderBook.Asks.MinPrice()
		}
		if quantityToTrade.Cmp(zero) > 0 {
			quote["order_id"] = strconv.FormatUint(orderBook.Item.NextOrderID, 10)
			quote["quantity"] = quantityToTrade.String()
			orderBook.Bids.InsertOrder(quote)
			orderInBook = quote
		}
	} else {
		maxPrice := orderBook.Bids.MaxPrice()
		for quantityToTrade.Cmp(zero) > 0 && orderBook.Bids.NotEmpty() && price.Cmp(maxPrice) <= 0 {
			bestPriceBids := orderBook.Bids.MaxPriceList()
			quantityToTrade, newTrades = orderBook.processOrderList(Bid, bestPriceBids, quantityToTrade, quote, verbose)
			trades = append(trades, newTrades...)
			maxPrice = orderBook.Bids.MaxPrice()
		}
		if quantityToTrade.Cmp(zero) > 0 {
			quote["order_id"] = strconv.FormatUint(orderBook.Item.NextOrderID, 10)
			quote["quantity"] = quantityToTrade.String()
			orderBook.Asks.InsertOrder(quote)
			orderInBook = quote
		}
	}
	return trades, orderInBook
}
func (orderBook *Orderbook) ProcessOrder(quote map[string]string, verbose bool) ([]map[string]string, map[string]string) {
	orderType := quote["type"]
	var orderInBook map[string]string
	var trades []map[string]string
	orderBook.UpdateTime()
	orderBook.Item.NextOrderID++
	if orderType == Market {
		trades = orderBook.processMarketOrder(quote, verbose)
	} else {
		trades, orderInBook = orderBook.processLimitOrder(quote, verbose)
	}
	orderBook.Save()
	return trades, orderInBook
}
func (orderBook *Orderbook) processOrderList(side string, orderList *OrderList, quantityStillToTrade *big.Int, quote map[string]string, verbose bool) (*big.Int, []map[string]string) {
	quantityToTrade := CloneBigInt(quantityStillToTrade)
	var trades []map[string]string
	zero := Zero()
	for orderList.Item.Length > 0 && quantityToTrade.Cmp(zero) > 0 {
		headOrder := orderList.GetOrder(orderList.Item.HeadOrder)
		if headOrder == nil {
			panic("headOrder is null")
		}
		tradedPrice := CloneBigInt(headOrder.Item.Price)
		var newBookQuantity *big.Int
		var tradedQuantity *big.Int
		if IsStrictlySmallerThan(quantityToTrade, headOrder.Item.Quantity) {
			tradedQuantity = CloneBigInt(quantityToTrade)
			newBookQuantity = Sub(headOrder.Item.Quantity, quantityToTrade)
			headOrder.UpdateQuantity(orderList, newBookQuantity, headOrder.Item.Timestamp)
			quantityToTrade = Zero()
		} else if IsEqual(quantityToTrade, headOrder.Item.Quantity) {
			tradedQuantity = CloneBigInt(quantityToTrade)
			if side == Bid {
				orderBook.Bids.RemoveOrder(headOrder)
			} else {
				orderBook.Asks.RemoveOrder(headOrder)
			}
			quantityToTrade = Zero()
		} else {
			tradedQuantity = CloneBigInt(headOrder.Item.Quantity)
			if side == Bid {
				orderBook.Bids.RemoveOrder(headOrder)
			} else {
				orderBook.Asks.RemoveOrderFromOrderList(headOrder, orderList)
			}
		}
		if verbose {
			fmt.Printf("TRADE: Timestamp - %d, Price - %s, Quantity - %s, TradeID - %s, Matching TradeID - %s\n", orderBook.Item.Timestamp, tradedPrice, tradedQuantity, headOrder.Item.TradeID, quote["trade_id"])
		}
		transactionRecord := make(map[string]string)
		transactionRecord["timestamp"] = strconv.FormatUint(orderBook.Item.Timestamp, 10)
		transactionRecord["price"] = tradedPrice.String()
		transactionRecord["quantity"] = tradedQuantity.String()
		trades = append(trades, transactionRecord)
	}
	return quantityToTrade, trades
}
func (orderBook *Orderbook) CancelOrder(side string, orderID uint64, price *big.Int) error {
	orderBook.UpdateTime()
	key := GetKeyFromBig(big.NewInt(int64(orderID)))
	var err error
	if side == Bid {
		order := orderBook.Bids.GetOrder(key, price)
		if order != nil {
			_, err = orderBook.Bids.RemoveOrder(order)
		}
	} else {
		order := orderBook.Asks.GetOrder(key, price)
		if order != nil {
			_, err = orderBook.Asks.RemoveOrder(order)
		}
	}
	return err
}
func (orderBook *Orderbook) UpdateOrder(quoteUpdate map[string]string) error {
	orderID, err := strconv.ParseUint(quoteUpdate["order_id"], 10, 64)
	if err == nil {
		price, ok := new(big.Int).SetString(quoteUpdate["price"], 10)
		if !ok {
			return fmt.Errorf("Price is not correct :%s", quoteUpdate["price"])
		}
		return orderBook.ModifyOrder(quoteUpdate, orderID, price)
	}
	return err
}
func (orderBook *Orderbook) ModifyOrder(quoteUpdate map[string]string, orderID uint64, price *big.Int) error {
	orderBook.UpdateTime()
	side := quoteUpdate["side"]
	quoteUpdate["order_id"] = strconv.FormatUint(orderID, 10)
	quoteUpdate["timestamp"] = strconv.FormatUint(orderBook.Item.Timestamp, 10)
	key := GetKeyFromBig(ToBigInt(quoteUpdate["order_id"]))
	if side == Bid {
		if orderBook.Bids.OrderExist(key, price) {
			return orderBook.Bids.UpdateOrder(quoteUpdate)
		}
	} else {
		if orderBook.Asks.OrderExist(key, price) {
			return orderBook.Asks.UpdateOrder(quoteUpdate)
		}
	}
	return nil
}
func (orderBook *Orderbook) VolumeAtPrice(side string, price *big.Int) *big.Int {
	volume := Zero()
	if side == Bid {
		if orderBook.Bids.PriceExist(price) {
			orderList := orderBook.Bids.PriceList(price)
			volume = CloneBigInt(orderList.Item.Volume)
		}
	} else {
		if orderBook.Asks.PriceExist(price) {
			orderList := orderBook.Asks.PriceList(price)
			volume = CloneBigInt(orderList.Item.Volume)
		}
	}
	return volume
}
