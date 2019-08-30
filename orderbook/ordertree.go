package orderbook

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

type OrderTreeItem struct {
	Volume        *big.Int
	NumOrders     uint64
	PriceTreeKey  []byte
	PriceTreeSize uint64
}
type OrderTree struct {
	PriceTree *RedBlackTreeExtended
	orderBook *Orderbook
	orderDB   *BatchDatabase
	slot      *big.Int
	Key       []byte
	Item      *OrderTreeItem
}

func NewOrderTree(orderDB *BatchDatabase, key []byte, orderBook *Orderbook) *OrderTree {
	priceTree := NewRedBlackTreeExtended(orderDB)
	item := &OrderTreeItem{Volume: Zero(), NumOrders: 0, PriceTreeSize: 0}
	slot := new(big.Int).SetBytes(key)
	orderTree := &OrderTree{orderDB: orderDB, PriceTree: priceTree, Key: key, slot: slot, Item: item, orderBook: orderBook}
	return orderTree
}
func (orderTree *OrderTree) Save() error {
	priceTreeRoot := orderTree.PriceTree.Root()
	if priceTreeRoot != nil {
		orderTree.Item.PriceTreeKey = priceTreeRoot.Key
		orderTree.Item.PriceTreeSize = orderTree.Depth()
	}
	return orderTree.orderDB.Put(orderTree.Key, orderTree.Item)
}
func (orderTree *OrderTree) Commit() error {
	err := orderTree.Save()
	if err == nil {
		err = orderTree.orderDB.Commit()
	}
	return err
}
func (orderTree *OrderTree) Restore() error {
	val, err := orderTree.orderDB.Get(orderTree.Key, orderTree.Item)
	if err == nil {
		orderTree.Item = val.(*OrderTreeItem)
		orderTree.PriceTree.SetRootKey(orderTree.Item.PriceTreeKey, orderTree.Item.PriceTreeSize)
	}
	return err
}
func (orderTree *OrderTree) String(startDepth int) string {
	tabs := strings.Repeat("\t", startDepth)
	return fmt.Sprintf("{\n\t%sMinPriceList: %s\n\t%sMaxPriceList: %s\n\t%sVolume: %v\n\t%sNumOrders: %d\n\t%sDepth: %d\n%s}", tabs, orderTree.MinPriceList().String(startDepth+1), tabs, orderTree.MaxPriceList().String(startDepth+1), tabs, orderTree.Item.Volume, tabs, orderTree.Item.NumOrders, tabs, orderTree.Depth(), tabs)
}
func (orderTree *OrderTree) Length() uint64 {
	return orderTree.Item.NumOrders
}
func (orderTree *OrderTree) NotEmpty() bool {
	return orderTree.Item.NumOrders > 0
}
func (orderTree *OrderTree) GetOrder(key []byte, price *big.Int) *Order {
	orderList := orderTree.PriceList(price)
	if orderList == nil {
		return nil
	}
	return orderList.GetOrder(key)
}
func (orderTree *OrderTree) getSlotFromPrice(price *big.Int) *big.Int {
	return Add(orderTree.slot, price)
}
func (orderTree *OrderTree) getKeyFromPrice(price *big.Int) []byte {
	orderListKey := orderTree.getSlotFromPrice(price)
	return GetKeyFromBig(orderListKey)
}
func (orderTree *OrderTree) PriceList(price *big.Int) *OrderList {
	key := orderTree.getKeyFromPrice(price)
	bytes, found := orderTree.PriceTree.Get(key)
	if found {
		orderList := orderTree.decodeOrderList(bytes)
		return orderList
	}
	return nil
}
func (orderTree *OrderTree) CreatePrice(price *big.Int) *OrderList {
	newList := NewOrderList(price, orderTree)
	newList.Save()
	orderTree.Save()
	return newList
}
func (orderTree *OrderTree) SaveOrderList(orderList *OrderList) error {
	value, err := orderTree.orderDB.EncodeToBytes(orderList.Item)
	if err != nil {
		fmt.Println(err)
		return err
	}
	if orderTree.orderDB.Debug {
		fmt.Printf("Save orderlist key %x, value :%x\n", orderList.Key, value)
	}
	return orderTree.PriceTree.Put(orderList.Key, value)
}
func (orderTree *OrderTree) Depth() uint64 {
	return orderTree.PriceTree.Size()
}
func (orderTree *OrderTree) RemovePrice(price *big.Int) {
	if orderTree.Depth() > 0 {
		orderListKey := orderTree.getKeyFromPrice(price)
		orderTree.PriceTree.Remove(orderListKey)
		orderTree.Save()
	}
}
func (orderTree *OrderTree) PriceExist(price *big.Int) bool {
	orderListKey := orderTree.getKeyFromPrice(price)
	found, _ := orderTree.PriceTree.Has(orderListKey)
	return found
}
func (orderTree *OrderTree) OrderExist(key []byte, price *big.Int) bool {
	orderList := orderTree.PriceList(price)
	if orderList == nil {
		return false
	}
	return orderList.OrderExist(key)
}
func (orderTree *OrderTree) InsertOrder(quote map[string]string) error {
	price := ToBigInt(quote["price"])
	var orderList *OrderList
	if !orderTree.PriceExist(price) {
		fmt.Println("CREATE price list", price.String())
		orderList = orderTree.CreatePrice(price)
	} else {
		orderList = orderTree.PriceList(price)
	}
	if orderList != nil {
		order := NewOrder(quote, orderList.Key)
		if orderList.OrderExist(order.Key) {
			orderTree.RemoveOrder(order)
		}
		orderList.AppendOrder(order)
		orderList.Save()
		orderList.SaveOrder(order)
		orderTree.Item.Volume = Add(orderTree.Item.Volume, order.Item.Quantity)
		orderTree.Item.NumOrders++
		return orderTree.Save()
	}
	return nil
}
func (orderTree *OrderTree) UpdateOrder(quote map[string]string) error {
	price := ToBigInt(quote["price"])
	orderList := orderTree.PriceList(price)
	if orderList == nil {
		orderList = orderTree.CreatePrice(price)
	}
	orderID := ToBigInt(quote["order_id"])
	key := GetKeyFromBig(orderID)
	order := orderList.GetOrder(key)
	originalQuantity := CloneBigInt(order.Item.Quantity)
	if !IsEqual(price, order.Item.Price) {
		orderList.RemoveOrder(order)
		if orderList.Item.Length == 0 {
			orderTree.RemovePrice(price)
		}
		orderTree.InsertOrder(quote)
	} else {
		quantity := ToBigInt(quote["quantity"])
		timestamp, _ := strconv.ParseUint(quote["timestamp"], 10, 64)
		order.UpdateQuantity(orderList, quantity, timestamp)
	}
	orderTree.Item.Volume = Add(orderTree.Item.Volume, Sub(order.Item.Quantity, originalQuantity))
	return orderTree.Save()
}
func (orderTree *OrderTree) RemoveOrderFromOrderList(order *Order, orderList *OrderList) error {
	err := orderList.RemoveOrder(order)
	if err != nil {
		return err
	}
	if orderList.Item.Length == 0 {
		orderTree.RemovePrice(order.Item.Price)
		fmt.Println("REMOVE price list", order.Item.Price.String())
	}
	orderTree.Item.Volume = Sub(orderTree.Item.Volume, order.Item.Quantity)
	orderTree.Item.NumOrders--
	return orderTree.Save()
}
func (orderTree *OrderTree) RemoveOrder(order *Order) (*OrderList, error) {
	var err error
	orderList := orderTree.PriceList(order.Item.Price)
	if orderList != nil {
		err = orderTree.RemoveOrderFromOrderList(order, orderList)
	}
	return orderList, err
}
func (orderTree *OrderTree) getOrderListItem(bytes []byte) *OrderListItem {
	item := &OrderListItem{}
	orderTree.orderDB.DecodeBytes(bytes, item)
	return item
}
func (orderTree *OrderTree) decodeOrderList(bytes []byte) *OrderList {
	item := orderTree.getOrderListItem(bytes)
	orderList := NewOrderListWithItem(item, orderTree)
	return orderList
}
func (orderTree *OrderTree) MaxPrice() *big.Int {
	if orderTree.Depth() > 0 {
		if bytes, found := orderTree.PriceTree.GetMax(); found {
			item := orderTree.getOrderListItem(bytes)
			if item != nil {
				return CloneBigInt(item.Price)
			}
		}
	}
	return Zero()
}
func (orderTree *OrderTree) MinPrice() *big.Int {
	if orderTree.Depth() > 0 {
		if bytes, found := orderTree.PriceTree.GetMin(); found {
			item := orderTree.getOrderListItem(bytes)
			if item != nil {
				return CloneBigInt(item.Price)
			}
		}
	}
	return Zero()
}
func (orderTree *OrderTree) MaxPriceList() *OrderList {
	if orderTree.Depth() > 0 {
		if bytes, found := orderTree.PriceTree.GetMax(); found {
			return orderTree.decodeOrderList(bytes)
		}
	}
	return nil
}
func (orderTree *OrderTree) MinPriceList() *OrderList {
	if orderTree.Depth() > 0 {
		if bytes, found := orderTree.PriceTree.GetMin(); found {
			return orderTree.decodeOrderList(bytes)
		}
	}
	return nil
}
