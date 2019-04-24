package tree

func PrintTree(b Tree) map[string]interface{} {
	result := make(map[string]interface{})
	var branchInfos []map[string]interface{}
	for _, v := range b.Branches() {
		branchInfos = append(branchInfos, PrintBranchInfo(v))
	}
	result["branches"] = branchInfos
	return result
}

func PrintBranchInfo(b Branch) map[string]interface{} {
	result := make(map[string]interface{})
	result["Id"] = b.Id()
	if b.Type() == Normal {
		result["Head"] = b.SprintHead()
		result["Tail"] = b.SprintTail()
		result["Root"] = b.Root().Id()
		var children []string
		for _, v := range b.(*branch).allChildren() {
			children = append(children, v.Id())
		}
		result["Children"] = children
	} else {
		result["Head"] = b.SprintHead()
	}
	return result
}