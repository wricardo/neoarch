package neoarch

import (
	"fmt"
	"strings"

	"log"
)

// ToStructurizrDSL outputs a Structurizr DSL representation of the entire design
// using hierarchical identifiers (e.g., "system.container.component") for nodes.
func (d *Design) ToStructurizrDSL() string {
	// 1) Build a lookup of nodes by ID.
	nodeByID := make(map[string]*Node)
	for _, n := range d.nodes {
		nodeByID[n.FullId()] = n
	}

	// 2) Build a parent->children map from BELONGS_TO relationships.
	parentChildren := map[string][]*Node{}
	for _, rel := range d.relationships {
		if rel.Type == RelBelongsTo {
			child := nodeByID[rel.StartID]
			if child != nil {
				parentChildren[rel.EndID] = append(parentChildren[rel.EndID], child)
			} else {
				log.Printf("Warning: Child node %s not found for relationship %v", rel.StartID, rel)
			}
		}
	}
	// parentChildren

	// 3) Identify the design (root) node by looking for NodeTypeDesign.
	var designNode *Node
	for _, n := range d.nodes {
		if n.NodeType == NodeTypeDesign {
			designNode = n
			break
		}
	}
	if designNode == nil {
		// If no design node is found, return a comment.
		return "// No design node found"
	}

	// 4) Build hierarchical short names.
	// For each node in the hierarchy, assign a short name based on its parent's name.
	// The design node (root) gets an empty short name so that its immediate children are top-level.

	// Use a breadth-first traversal starting from the design node.
	queue := []*Node{designNode}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		children := parentChildren[current.FullId()]
		for _, child := range children {
			// localName := makeId(child.ID)
			queue = append(queue, child)
		}
	}

	// 5) Emit the DSL.
	sb := &stringBuilderWithIndent{}
	sb.WriteLinef(`workspace "%s" "%s" {`, d.Name, d.Description)
	sb.Indent()
	sb.WriteLine(`!identifiers hierarchical`)
	sb.WriteLine("")

	// Emit the model block.
	sb.WriteLine("model {")
	sb.Indent()
	// Recursively emit each top-level node that belongs to the root (design) node.
	for _, top := range parentChildren[designNode.FullId()] {
		emitNodeDSL(sb, top, parentChildren)
	}
	sb.WriteLine("")
	// Emit relationships (excluding BELONGS_TO and IMPLIED_USE).
	for _, rel := range d.relationships {
		if rel.Type == RelBelongsTo || rel.Type == RelImpliedUse {
			continue
		}
		// startShort := shortNameMap[rel.StartID]
		// endShort := shortNameMap[rel.EndID]
		// if startShort == "" || endShort == "" {
		// 	continue
		// }
		// need to build a full name with the short names from these nodes
		// startNode := nodeByID[rel.StartID]
		// startTmp := startNode.FullId()
		// startFinal := strings.Split(startTmp, ".")
		// for i := 0; i < len(startFinal); i++ {
		// 	startFinal[i] = makeId(startFinal[i])
		// }

		// endNode := nodeByID[rel.EndID]
		// endTmp := endNode.FullId()
		// endFinal := strings.Split(endTmp, ".")
		// for i := 0; i < len(endFinal); i++ {
		// 	endFinal[i] = makeId(endFinal[i])
		// }
		// // startFull := strings.Join(startFinal, ".")
		// startFull := strings.Join(startFinal, ".")
		// endFull := strings.Join(endFinal, ".")

		sb.WriteLinef(`%s -> %s "%s"`, rel.StartID, rel.EndID, escapeQuotes(rel.Description))
		// sb.WriteLinef(`%s -> %s "%s"`, startShort, endShort, escapeQuotes(rel.Description))
	}
	sb.Dedent()
	sb.WriteLine("}") // end model

	// Emit a simple views block.
	sb.WriteLine("")
	sb.WriteLine("views {")
	sb.Indent()
	// For each top-level system (child of the design node), define systemContext and container views.
	for _, sys := range parentChildren[designNode.ID] {
		if sys.NodeType == NodeTypeSystem {
			// sysShort := makeId(sys.ID)
			sysShort := sys.ID
			sysName := sanitizeQuotes(sys.Name)
			sb.WriteLinef(`systemContext %s "system_context_%s" {`, sysShort, sysName)
			sb.Indent()
			sb.WriteLine("include *")
			sb.WriteLine("autolayout lr")
			sb.Dedent()
			sb.WriteLine("}")
			sb.WriteLine("")
			sb.WriteLinef(`container %s "container_%s" {`, sysShort, sysName)
			sb.Indent()
			sb.WriteLine("include *")
			sb.WriteLine("autolayout lr")
			sb.Dedent()
			sb.WriteLine("}")
			sb.WriteLine("")
		}
	}
	sb.Dedent()
	sb.WriteLine("}") // end views

	sb.Dedent()
	sb.WriteLine("}") // end workspace

	return sb.String()
}

// emitNodeDSL recursively emits DSL for a given node based on its type.
func emitNodeDSL(sb *stringBuilderWithIndent, n *Node,
	parentChildren map[string][]*Node,
) {
	// thisShort := makeId(n.Name)
	thisName := sanitizeQuotes(n.Name)
	thisDesc := sanitizeQuotes(n.Description)

	switch n.NodeType {
	case NodeTypePerson:
		if len(n.Tags) == 0 {
			sb.WriteLinef(`%s = person "%s" "%s"`, n.ID, thisName, thisDesc)
		} else {
			sb.WriteLinef(`%s = person "%s" "%s" {`, n.ID, thisName, thisDesc)
			sb.Indent()
			emitTags(sb, n.Tags)
			sb.Dedent()
			sb.WriteLine("}")
		}
	case NodeTypeSystem:
		children := parentChildren[n.ID]
		if len(children) == 0 && len(n.Tags) == 0 {
			sb.WriteLinef(`%s = softwareSystem "%s" "%s"`, n.ID, thisName, thisDesc)
		} else {
			sb.WriteLinef(`%s = softwareSystem "%s" "%s" {`, n.ID, thisName, thisDesc)
			sb.Indent()
			if len(n.Tags) > 0 {
				emitTags(sb, n.Tags)
			}
			for _, c := range children {
				emitNodeDSL(sb, c, parentChildren)
			}
			sb.Dedent()
			sb.WriteLine("}")
		}
	case NodeTypeContainer:
		children := parentChildren[n.FullId()]
		if len(children) == 0 && len(n.Tags) == 0 {
			sb.WriteLinef(`%s = container "%s" "%s"`, n.ID, thisName, thisDesc)
		} else {
			sb.WriteLinef(`%s = container "%s" "%s" {`, n.ID, thisName, thisDesc)
			sb.Indent()
			if len(n.Tags) > 0 {
				emitTags(sb, n.Tags)
			}
			for _, c := range children {
				emitNodeDSL(sb, c, parentChildren)
			}
			sb.Dedent()
			sb.WriteLine("}")
		}
	case NodeTypeComponent:
		if len(n.Tags) == 0 {
			sb.WriteLinef(`%s = component "%s" "%s"`, n.ID, thisName, thisDesc)
		} else {
			sb.WriteLinef(`%s = component "%s" "%s" {`, n.ID, thisName, thisDesc)
			sb.Indent()
			emitTags(sb, n.Tags)
			sb.Dedent()
			sb.WriteLine("}")
		}
	case NodeTypeDesign:
		// The design (root) node is not emitted.
		return
	}
}

// emitTags outputs the list of tags for a node.
func emitTags(sb *stringBuilderWithIndent, tags []string) {
	if len(tags) == 0 {
		return
	}
	sb.Write("\ntags")
	for _, tag := range tags {
		sb.Write(fmt.Sprintf(` "%s"`, sanitizeQuotes(tag)))
	}
	sb.WriteLine("")
}

// sanitizeQuotes escapes double quotes for DSL output.
func sanitizeQuotes(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// escapeQuotes is an alias for sanitizeQuotes.
func escapeQuotes(s string) string {
	return sanitizeQuotes(s)
}

// stringBuilderWithIndent is a helper for assembling strings with indentation.
type stringBuilderWithIndent struct {
	lines  []string
	indent int
}

func (s *stringBuilderWithIndent) Indent() {
	s.indent++
}

func (s *stringBuilderWithIndent) Dedent() {
	if s.indent > 0 {
		s.indent--
	}
}

func (s *stringBuilderWithIndent) Write(str string) {
	if len(s.lines) == 0 {
		s.lines = append(s.lines, strings.Repeat("    ", s.indent)+str)
	} else {
		// Append to the last line.
		lastIndex := len(s.lines) - 1
		s.lines[lastIndex] += str
	}
}

func (s *stringBuilderWithIndent) WriteLine(str string) {
	s.lines = append(s.lines, strings.Repeat("    ", s.indent)+str)
}

func (s *stringBuilderWithIndent) WriteLinef(format string, args ...any) {
	s.lines = append(s.lines, strings.Repeat("    ", s.indent)+fmt.Sprintf(format, args...))
}

func (s *stringBuilderWithIndent) String() string {
	return strings.Join(s.lines, "\n")
}
