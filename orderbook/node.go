package orderbook

import (
	"fmt"
)

type KeyMeta struct {
	Left   []byte
	Right  []byte
	Parent []byte
}
type Item struct {
	Keys  *KeyMeta
	Value []byte
	Color bool
}
type Node struct {
	Key  []byte
	Item *Item
}

const (
	black, red bool = true, false
)

func (keys *KeyMeta) String(tree *Tree) string {
	return fmt.Sprintf("L: %v, P: %v, R: %v", tree.FormatBytes(keys.Left), tree.FormatBytes(keys.Parent), tree.FormatBytes(keys.Right))
}
func (node *Node) String(tree *Tree) string {
	if node == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v -> %x, (%v)\n", tree.FormatBytes(node.Key), node.Value(), node.Item.Keys.String(tree))
}
func (node *Node) maximumNode(tree *Tree) *Node {
	newNode := node
	if newNode == nil {
		return newNode
	}
	for !tree.IsEmptyKey(node.RightKey()) {
		newNode = newNode.Right(tree)
	}
	return newNode
}
func (node *Node) LeftKey(keys ...[]byte) []byte {
	if node == nil || node.Item == nil || node.Item.Keys == nil {
		return nil
	}
	if len(keys) == 1 {
		node.Item.Keys.Left = keys[0]
	}
	return node.Item.Keys.Left
}
func (node *Node) RightKey(keys ...[]byte) []byte {
	if node == nil || node.Item == nil || node.Item.Keys == nil {
		return nil
	}
	if len(keys) == 1 {
		node.Item.Keys.Right = keys[0]
	}
	return node.Item.Keys.Right
}
func (node *Node) ParentKey(keys ...[]byte) []byte {
	if node == nil || node.Item == nil || node.Item.Keys == nil {
		return nil
	}
	if len(keys) == 1 {
		node.Item.Keys.Parent = keys[0]
	}
	return node.Item.Keys.Parent
}
func (node *Node) Left(tree *Tree) *Node {
	key := node.LeftKey()
	newNode, err := tree.GetNode(key)
	if err != nil {
		fmt.Println(err)
	}
	return newNode
}
func (node *Node) Right(tree *Tree) *Node {
	key := node.RightKey()
	newNode, err := tree.GetNode(key)
	if err != nil {
		fmt.Println(err)
	}
	return newNode
}
func (node *Node) Parent(tree *Tree) *Node {
	key := node.ParentKey()
	newNode, err := tree.GetNode(key)
	if err != nil {
		fmt.Println(err)
	}
	return newNode
}
func (node *Node) Value() []byte {
	return node.Item.Value
}
func (node *Node) grandparent(tree *Tree) *Node {
	if node != nil && !tree.IsEmptyKey(node.ParentKey()) {
		return node.Parent(tree).Parent(tree)
	}
	return nil
}
func (node *Node) uncle(tree *Tree) *Node {
	if node == nil || tree.IsEmptyKey(node.ParentKey()) {
		return nil
	}
	parent := node.Parent(tree)
	return parent.sibling(tree)
}
func (node *Node) sibling(tree *Tree) *Node {
	if node == nil || tree.IsEmptyKey(node.ParentKey()) {
		return nil
	}
	parent := node.Parent(tree)
	if tree.Comparator(node.Key, parent.LeftKey()) == 0 {
		return parent.Right(tree)
	}
	return parent.Left(tree)
}
