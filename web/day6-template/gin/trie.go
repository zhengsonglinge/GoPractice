package gin

import "strings"

/*
实现前缀树用来实现两个功能：
- 参数匹配:。例如 /p/:lang/doc，可以匹配 /p/c/doc 和 /p/go/doc。
- 通配*。例如 /static/*filepath，可以匹配/static/fav.ico，也可以匹配/static/js/jQuery.js，
  这种模式常用于静态服务器，能够递归地匹配子路径。
*/
type node struct {
	pattern  string  // 待匹配路，如 /p/:lang
	part     string  //路由中的一部，如 :lang
	children []*node //子节点
	isWild   bool    //是否模糊匹配，part 含有 : 或 * 时为 true
}

// 第一个匹配成功的节点，用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 所有匹配成功的节点，用于查找
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

// 递归插入前缀树节点
func (n *node) insert(pattern string, parts []string, height int) {
	// 当当前前缀树节点的高度等于路径长度时，表示到达叶子节点
	// 只有叶子节点才设置 pattern 字段，用来标识到达叶子节点，即路径的末尾
	if len(parts) == height {
		n.pattern = pattern
		return
	}
	part := parts[height]
	// 查找当前前缀树是否已经注册过 part 节点，如果没有则创建
	child := n.matchChild(part)
	if child == nil {
		child = &node{
			part:   part,
			isWild: part[0] == ':' || part[0] == '*',
		}
		n.children = append(n.children, child)
	}
	// 递归找到最后的叶子节点
	child.insert(pattern, parts, height+1)
}

// 递归查找前缀树节点
// 返回值是一个与路径相同的叶子节点
func (n *node) search(parts []string, height int) *node {
	// 如果到达路径末尾或当前节点含有通配符 * 则结束递归
	// 如果当前节点 pattern 不为空，说明到达了叶子节点，返回当前节点，否则返回 nil
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	// 找出与当前节点的 part 字段相同或含有 : 或 * 的子节点
	part := parts[height]
	children := n.matchChildren(part)

	// 递归遍历符合条件的子节点，如果有叶子节点的路径与查询路径相同则返回叶子节点
	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}
	return nil
}
