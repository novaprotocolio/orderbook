package orderbook

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
)

type OrderItem struct {
	Timestamp uint64
	Quantity  *big.Int
	Price     *big.Int
	TradeID   string
	NextOrder []byte
	PrevOrder []byte
	OrderList []byte
}
type Order struct {
	Item *OrderItem
	Key  []byte
}

func (order *Order) String() string {
	return fmt.Sprintf("orderID : %s, price: %s, quantity :%s, tradeID: %s", new(big.Int).SetBytes(order.Key), order.Item.Price, order.Item.Quantity, order.Item.TradeID)
}
func (order *Order) GetNextOrder(orderList *OrderList) *Order {
	nextOrder := orderList.GetOrder(order.Item.NextOrder)
	return nextOrder
}
func (order *Order) GetPrevOrder(orderList *OrderList) *Order {
	prevOrder := orderList.GetOrder(order.Item.PrevOrder)
	return prevOrder
}
func NewOrder(quote map[string]string, orderList []byte) *Order {
	timestamp, _ := strconv.ParseUint(quote["timestamp"], 10, 64)
	quantity := ToBigInt(quote["quantity"])
	price := ToBigInt(quote["price"])
	orderID := ToBigInt(quote["order_id"])
	key := GetKeyFromBig(orderID)
	tradeID := quote["trade_id"]
	orderItem := &OrderItem{Timestamp: timestamp, Quantity: quantity, Price: price, TradeID: tradeID, NextOrder: EmptyKey(), PrevOrder: EmptyKey(), OrderList: orderList}
	order := &Order{Key: key, Item: orderItem}
	return order
}
func (order *Order) UpdateQuantity(orderList *OrderList, newQuantity *big.Int, newTimestamp uint64) {
	if newQuantity.Cmp(order.Item.Quantity) > 0 && !bytes.Equal(orderList.Item.TailOrder, order.Key) {
		orderList.MoveToTail(order)
	}
	orderList.Item.Volume = Sub(orderList.Item.Volume, Sub(order.Item.Quantity, newQuantity))
	order.Item.Timestamp = newTimestamp
	order.Item.Quantity = CloneBigInt(newQuantity)
	fmt.Println("QUANTITY", order.Item.Quantity.String())
	orderList.SaveOrder(order)
	orderList.Save()
}
