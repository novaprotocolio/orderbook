package orderbook

import (
	"bytes"
	"fmt"
)

type Tree struct {
	db          *BatchDatabase
	rootKey     []byte
	size        uint64
	Comparator  Comparator
	FormatBytes FormatBytes
}

func NewWith(comparator Comparator, db *BatchDatabase) *Tree {
	tree := &Tree{Comparator: comparator, db: db}
	return tree
}
func NewWithBytesComparator(db *BatchDatabase) *Tree {
	return NewWith(bytes.Compare, db)
}
func (tree *Tree) Root() *Node {
	root, _ := tree.GetNode(tree.rootKey)
	return root
}
func (tree *Tree) IsEmptyKey(key []byte) bool {
	return tree.db.IsEmptyKey(key)
}
func (tree *Tree) SetRootKey(key []byte, size uint64) {
	tree.rootKey = key
	tree.size = size
}
func (tree *Tree) Put(key []byte, value []byte) error {
	var insertedNode *Node
	if tree.IsEmptyKey(tree.rootKey) {
		item := &Item{Value: value, Color: red, Keys: &KeyMeta{}}
		tree.rootKey = key
		insertedNode = &Node{Key: key, Item: item}
	} else {
		node := tree.Root()
		loop := true
		for loop {
			compare := tree.Comparator(key, node.Key)
			switch {
			case compare == 0:
				node.Item.Value = value
				tree.Save(node)
				return nil
			case compare < 0:
				if tree.IsEmptyKey(node.LeftKey()) {
					node.LeftKey(key)
					tree.Save(node)
					item := &Item{Value: value, Color: red, Keys: &KeyMeta{}}
					nodeLeft := &Node{Key: key, Item: item}
					insertedNode = nodeLeft
					loop = false
				} else {
					node = node.Left(tree)
				}
			case compare > 0:
				if tree.IsEmptyKey(node.RightKey()) {
					node.RightKey(key)
					tree.Save(node)
					item := &Item{Value: value, Color: red, Keys: &KeyMeta{}}
					nodeRight := &Node{Key: key, Item: item}
					insertedNode = nodeRight
					loop = false
				} else {
					node = node.Right(tree)
				}
			}
		}
		insertedNode.ParentKey(node.Key)
		tree.Save(insertedNode)
	}
	tree.insertCase1(insertedNode)
	tree.Save(insertedNode)
	tree.size++
	return nil
}
func (tree *Tree) GetNode(key []byte) (*Node, error) {
	item := &Item{}
	val, err := tree.db.Get(key, item)
	if err != nil || val == nil {
		return nil, err
	}
	return &Node{Key: key, Item: val.(*Item)}, err
}
func (tree *Tree) Has(key []byte) (bool, error) {
	return tree.db.Has(key)
}
func (tree *Tree) Get(key []byte) (value []byte, found bool) {
	node, err := tree.GetNode(key)
	if err != nil {
		return nil, false
	}
	if node != nil {
		return node.Item.Value, true
	}
	return nil, false
}
func (tree *Tree) Remove(key []byte) {
	var child *Node
	node, err := tree.GetNode(key)
	if err != nil || node == nil {
		return
	}
	var left, right *Node = nil, nil
	if !tree.IsEmptyKey(node.LeftKey()) {
		left = node.Left(tree)
	}
	if !tree.IsEmptyKey(node.RightKey()) {
		right = node.Right(tree)
	}
	if left != nil && right != nil {
		node = left.maximumNode(tree)
	}
	if left == nil || right == nil {
		if right == nil {
			child = left
		} else {
			child = right
		}
		if node.Item.Color == black {
			node.Item.Color = nodeColor(child)
			tree.Save(node)
			tree.deleteCase1(node)
		}
		tree.replaceNode(node, child)
		if tree.IsEmptyKey(node.ParentKey()) && child != nil {
			child.Item.Color = black
			tree.Save(child)
		}
	}
	tree.size--
}
func (tree *Tree) Empty() bool {
	return tree.size == 0
}
func (tree *Tree) Size() uint64 {
	return tree.size
}
func (tree *Tree) Keys() [][]byte {
	keys := make([][]byte, tree.size)
	it := tree.Iterator()
	for i := 0; it.Next(); i++ {
		keys[i] = it.Key()
	}
	return keys
}
func (tree *Tree) Values() [][]byte {
	values := make([][]byte, tree.size)
	it := tree.Iterator()
	for i := 0; it.Next(); i++ {
		values[i] = it.Value()
	}
	return values
}
func (tree *Tree) Left() *Node {
	var parent *Node
	current := tree.Root()
	for current != nil {
		parent = current
		current = current.Left(tree)
	}
	return parent
}
func (tree *Tree) Right() *Node {
	var parent *Node
	current := tree.Root()
	for current != nil {
		parent = current
		current = current.Right(tree)
	}
	return parent
}
func (tree *Tree) Floor(key []byte) (floor *Node, found bool) {
	found = false
	node := tree.Root()
	for node != nil {
		compare := tree.Comparator(key, node.Key)
		switch {
		case compare == 0:
			return node, true
		case compare < 0:
			node = node.Left(tree)
		case compare > 0:
			floor, found = node, true
			node = node.Right(tree)
		}
	}
	if found {
		return floor, true
	}
	return nil, false
}
func (tree *Tree) Ceiling(key []byte) (ceiling *Node, found bool) {
	found = false
	node := tree.Root()
	for node != nil {
		compare := tree.Comparator(key, node.Key)
		switch {
		case compare == 0:
			return node, true
		case compare < 0:
			ceiling, found = node, true
			node = node.Left(tree)
		case compare > 0:
			node = node.Right(tree)
		}
	}
	if found {
		return ceiling, true
	}
	return nil, false
}
func (tree *Tree) Clear() {
	tree.rootKey = EmptyKey()
	tree.size = 0
}
func (tree *Tree) String() string {
	str := fmt.Sprintf("RedBlackTree, size: %d\n", tree.size)
	output(tree, tree.Root(), "", true, &str)
	return str
}
func output(tree *Tree, node *Node, prefix string, isTail bool, str *string) {
	if node == nil {
		return
	}
	if !tree.IsEmptyKey(node.RightKey()) {
		newPrefix := prefix
		if isTail {
			newPrefix += "│   "
		} else {
			newPrefix += "    "
		}
		output(tree, node.Right(tree), newPrefix, false, str)
	}
	*str += prefix
	if isTail {
		*str += "└── "
	} else {
		*str += "┌── "
	}
	if tree.FormatBytes != nil {
		*str += node.String(tree) + "\n"
	} else {
		*str += string(node.Key) + "\n"
	}
	if !tree.IsEmptyKey(node.LeftKey()) {
		newPrefix := prefix
		if isTail {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
		output(tree, node.Left(tree), newPrefix, true, str)
	}
}
func (tree *Tree) rotateLeft(node *Node) {
	right := node.Right(tree)
	tree.replaceNode(node, right)
	node.RightKey(right.LeftKey())
	if !tree.IsEmptyKey(right.LeftKey()) {
		rightLeft := right.Left(tree)
		rightLeft.ParentKey(node.Key)
		tree.Save(rightLeft)
	}
	right.LeftKey(node.Key)
	node.ParentKey(right.Key)
	tree.Save(node)
	tree.Save(right)
}
func (tree *Tree) rotateRight(node *Node) {
	left := node.Left(tree)
	tree.replaceNode(node, left)
	node.LeftKey(left.RightKey())
	if !tree.IsEmptyKey(left.RightKey()) {
		leftRight := left.Right(tree)
		leftRight.ParentKey(node.Key)
		tree.Save(leftRight)
	}
	left.RightKey(node.Key)
	node.ParentKey(left.Key)
	tree.Save(node)
	tree.Save(left)
}
func (tree *Tree) replaceNode(old *Node, new *Node) {
	var newKey []byte
	if new == nil {
		newKey = EmptyKey()
	} else {
		newKey = new.Key
	}
	if tree.IsEmptyKey(old.ParentKey()) {
		tree.rootKey = newKey
	} else {
		oldParent := old.Parent(tree)
		if tree.Comparator(old.Key, oldParent.LeftKey()) == 0 {
			oldParent.LeftKey(newKey)
		} else {
			oldParent.RightKey(newKey)
		}
		tree.Save(oldParent)
	}
	if new != nil {
		new.ParentKey(old.ParentKey())
		tree.Save(new)
	}
}
func (tree *Tree) insertCase1(node *Node) {
	if tree.IsEmptyKey(node.ParentKey()) {
		node.Item.Color = black
	} else {
		tree.insertCase2(node)
	}
}
func (tree *Tree) insertCase2(node *Node) {
	parent := node.Parent(tree)
	if nodeColor(parent) == black {
		return
	}
	tree.insertCase3(node)
}
func (tree *Tree) insertCase3(node *Node) {
	parent := node.Parent(tree)
	uncle := node.uncle(tree)
	if nodeColor(uncle) == red {
		parent.Item.Color = black
		uncle.Item.Color = black
		tree.Save(uncle)
		tree.Save(parent)
		grandparent := parent.Parent(tree)
		tree.assertNotNull(grandparent, "grant parent")
		grandparent.Item.Color = red
		tree.insertCase1(grandparent)
		tree.Save(grandparent)
	} else {
		tree.insertCase4(node)
	}
}
func (tree *Tree) insertCase4(node *Node) {
	parent := node.Parent(tree)
	grandparent := parent.Parent(tree)
	tree.assertNotNull(grandparent, "grant parent")
	if tree.Comparator(node.Key, parent.RightKey()) == 0 && tree.Comparator(parent.Key, grandparent.LeftKey()) == 0 {
		tree.rotateLeft(parent)
		node = node.Left(tree)
	} else if tree.Comparator(node.Key, parent.LeftKey()) == 0 && tree.Comparator(parent.Key, grandparent.RightKey()) == 0 {
		tree.rotateRight(parent)
		node = node.Right(tree)
	}
	tree.insertCase5(node)
}
func (tree *Tree) assertNotNull(node *Node, name string) {
	if node == nil {
		panic(fmt.Sprintf("%s is nil\n", name))
	}
}
func (tree *Tree) insertCase5(node *Node) {
	parent := node.Parent(tree)
	parent.Item.Color = black
	tree.Save(parent)
	grandparent := parent.Parent(tree)
	tree.assertNotNull(grandparent, "grant parent")
	grandparent.Item.Color = red
	tree.Save(grandparent)
	if tree.Comparator(node.Key, parent.LeftKey()) == 0 && tree.Comparator(parent.Key, grandparent.LeftKey()) == 0 {
		tree.rotateRight(grandparent)
	} else if tree.Comparator(node.Key, parent.RightKey()) == 0 && tree.Comparator(parent.Key, grandparent.RightKey()) == 0 {
		tree.rotateLeft(grandparent)
	}
}
func (tree *Tree) Save(node *Node) error {
	return tree.db.Put(node.Key, node.Item)
}
func (tree *Tree) Commit() error {
	return tree.db.Commit()
}
func (tree *Tree) deleteCase1(node *Node) {
	if tree.IsEmptyKey(node.ParentKey()) {
		return
	}
	tree.deleteCase2(node)
}
func (tree *Tree) deleteCase2(node *Node) {
	parent := node.Parent(tree)
	sibling := node.sibling(tree)
	if nodeColor(sibling) == red {
		parent.Item.Color = red
		sibling.Item.Color = black
		tree.Save(parent)
		tree.Save(sibling)
		if tree.Comparator(node.Key, parent.LeftKey()) == 0 {
			tree.rotateLeft(parent)
		} else {
			tree.rotateRight(parent)
		}
	}
	tree.deleteCase3(node)
}
func (tree *Tree) deleteCase3(node *Node) {
	parent := node.Parent(tree)
	sibling := node.sibling(tree)
	siblingLeft := sibling.Left(tree)
	siblingRight := sibling.Right(tree)
	if nodeColor(parent) == black && nodeColor(sibling) == black && nodeColor(siblingLeft) == black && nodeColor(siblingRight) == black {
		sibling.Item.Color = red
		tree.Save(sibling)
		tree.deleteCase1(parent)
		if tree.db.Debug {
			fmt.Printf("delete node,  key: %x, parentKey :%x\n", node.Key, parent.Key)
		}
		tree.deleteNode(node, false)
	} else {
		tree.deleteCase4(node)
	}
}
func (tree *Tree) deleteCase4(node *Node) {
	parent := node.Parent(tree)
	sibling := node.sibling(tree)
	siblingLeft := sibling.Left(tree)
	siblingRight := sibling.Right(tree)
	if nodeColor(parent) == red && nodeColor(sibling) == black && nodeColor(siblingLeft) == black && nodeColor(siblingRight) == black {
		sibling.Item.Color = red
		parent.Item.Color = black
		tree.Save(sibling)
		tree.Save(parent)
	} else {
		tree.deleteCase5(node)
	}
}
func (tree *Tree) deleteCase5(node *Node) {
	parent := node.Parent(tree)
	sibling := node.sibling(tree)
	siblingLeft := sibling.Left(tree)
	siblingRight := sibling.Right(tree)
	if tree.Comparator(node.Key, parent.LeftKey()) == 0 && nodeColor(sibling) == black && nodeColor(siblingLeft) == red && nodeColor(siblingRight) == black {
		sibling.Item.Color = red
		siblingLeft.Item.Color = black
		tree.Save(sibling)
		tree.Save(siblingLeft)
		tree.rotateRight(sibling)
	} else if tree.Comparator(node.Key, parent.RightKey()) == 0 && nodeColor(sibling) == black && nodeColor(siblingRight) == red && nodeColor(siblingLeft) == black {
		sibling.Item.Color = red
		siblingRight.Item.Color = black
		tree.Save(sibling)
		tree.Save(siblingRight)
		tree.rotateLeft(sibling)
	}
	tree.deleteCase6(node)
}
func (tree *Tree) deleteCase6(node *Node) {
	parent := node.Parent(tree)
	sibling := node.sibling(tree)
	siblingLeft := sibling.Left(tree)
	siblingRight := sibling.Right(tree)
	sibling.Item.Color = nodeColor(parent)
	parent.Item.Color = black
	tree.Save(sibling)
	tree.Save(parent)
	if tree.Comparator(node.Key, parent.LeftKey()) == 0 && nodeColor(siblingRight) == red {
		siblingRight.Item.Color = black
		tree.Save(siblingRight)
		tree.rotateLeft(parent)
	} else if nodeColor(siblingLeft) == red {
		siblingLeft.Item.Color = black
		tree.Save(siblingLeft)
		tree.rotateRight(parent)
	}
	tree.deleteNode(node, false)
}
func nodeColor(node *Node) bool {
	if node == nil {
		return black
	}
	return node.Item.Color
}
func (tree *Tree) deleteNode(node *Node, force bool) {
	tree.db.Delete(node.Key, force)
}
