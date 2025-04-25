// Package neoarch provides a domain-specific language for modeling C4 architecture
// and storing it in a Neo4j database. It allows creating and connecting architecture
// elements such as Persons, Systems, Containers, and Components, following the
// C4 model principles (https://c4model.com).
package neoarch

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// -----------------------------------------------------------------------------
// Enumerations & Basic Types
// -----------------------------------------------------------------------------

// NodeType is an enum-like type for different C4 elements.
type NodeType string

const (
	NodeTypeUnknown   NodeType = "Unknown"
	NodeTypeDesign    NodeType = "Design"
	NodeTypePerson    NodeType = "Person"
	NodeTypeSystem    NodeType = "System"
	NodeTypeContainer NodeType = "Container"
	NodeTypeComponent NodeType = "Component"
)

// RelationshipType is a type for naming relationships
type RelationshipType string

const (
	RelUses          RelationshipType = "USES"
	RelImpliedUse    RelationshipType = "IMPLIED_USE"
	RelBelongsTo     RelationshipType = "BELONGS_TO"
	RelInteractsWith RelationshipType = "INTERACTS_WITH"
)

// Relationship represents a direction from "start" to "end" with a type & description.
type Relationship struct {
	StartID     string
	EndID       string
	Type        RelationshipType
	Description string
}

// INode defines an interface for objects that can be identified uniquely in the design.
// Any type that implements this interface can be used as a node in relationships.
type INode interface {
	GetID() string
	FullName() string
	FullId() string
}

// -----------------------------------------------------------------------------
// Core Node & Derived Types
// -----------------------------------------------------------------------------

// Node is the shared struct for all C4 elements.
type Node struct {
	ID          string   // Unique identifier (could be the "name")
	Name        string   // Display name
	Labels      []string // Arbitrary extra labels that will be added to the node in addition to the node type
	Description string   // Brief description
	NodeType    NodeType // e.g. Person, System, Container, Component
	Tags        []string // Arbitrary tags
	IsExternal  bool     // For marking external nodes
	design      *Design  // Link back to the containing Design
	ParentNode  INode    // Parent node (if any)
}

func NewNodeWithIdAndParent(id string, parent INode, design *Design, name, description string, nodeType NodeType) *Node {
	n := &Node{
		ID:          id,
		Name:        name,
		Description: description,
		NodeType:    nodeType,
		ParentNode:  parent,
		design:      design,
	}
	if parent != nil {
		n.ID = parent.GetID() + "." + n.ID
	}

	return n
}

func NewNodeWithParent(parent INode, design *Design, name, description string, nodeType NodeType) *Node {
	return NewNodeWithIdAndParent(name, parent, design, name, description, nodeType)
}

func NewNode(name, description string, nodeType NodeType) *Node {
	return NewNodeWithIdAndParent(name, nil, nil, name, description, nodeType)
}

func (n *Node) AddLabel(label string) *Node {
	n.Labels = append(n.Labels, label)
	return n
}

func (n *Node) FullId() string {
	if n == nil {
		return ""
	}
	if n.ParentNode != nil {
		return n.ParentNode.FullId() + "." + n.ID
	}
	return n.ID
}

func (n *Node) FullName() string {
	if n.ParentNode != nil {
		return n.ParentNode.FullName() + "." + n.Name
	}
	return n.Name
}

// GetID returns the ID of the node.
func (n *Node) GetID() string {
	return n.ID
}

func (n *Node) Tag(tag string) {
	n.Tags = append(n.Tags, tag)
}
func (n *Node) External() {
	n.IsExternal = true
}
func (n *Node) Internal() {
	n.IsExternal = false
}

// Tag appends a tag to the Person.
func (p *Person) Tag(tag string) *Person {
	p.Node.Tag(tag)
	return p
}

func (p *Person) External() *Person {
	p.Node.External()
	return p
}

func (p *Person) Internal() *Person {
	p.Node.Internal()
	return p
}

// Tag appends a tag to the Container.
func (c *Container) Tag(tag string) *Container {
	c.Node.Tag(tag)
	return c
}

func (n *Container) AddLabel(label string) *Container {
	n.Node.AddLabel(label)
	return n
}

func (c *Container) External() *Container {
	c.Node.External()
	return c
}

func (c *Container) Internal() *Container {
	c.Node.Internal()
	return c
}

// Tag appends a tag to the Component.
func (c *Component) Tag(tag string) *Component {
	c.Node.Tag(tag)
	return c
}

func (c *Component) External() *Component {
	c.Node.External()
	return c
}

func (c *Component) Internal() *Component {
	c.Node.Internal()
	return c
}

// -----------------------------------------------------------------------------
// DSL Structures: Person, System, Container, Component
// Each is basically a wrapper around Node with chainable methods
// -----------------------------------------------------------------------------

// Person represents a "Person" node in C4.
type Person struct {
	*Node
	design *Design // Link back to the parent design
}

// InteractsWith creates a "INTERACTS_WITH" relationship from this person to another person.
func (p *Person) InteractsWith(other *Person, description string) *Person {
	p.design.addRelationship(p, other, RelInteractsWith, description)
	return p
}

func (p *Person) Uses(n INode, description string) *Person {
	p.design.addRelationship(p, n, RelUses, description)
	return p
}

// Person can also use "UsedBy" if you want to invert direction, but here we
// only define InteractsWith, as per your example usage.

// -----------------------------------------------------------------------------

// System represents a "System" node in C4.
type System struct {
	*Node
	design *Design // Link back to the parent design
}

func (s *System) ImpliedUse(p INode, description string) *System {
	// s uses p (implied)
	s.design.addRelationship(s, p, RelImpliedUse, description)
	return s
}

// ImpliedUsedBy creates an implied usage relationship from a person to this system.
func (s *System) ImpliedUsedBy(p INode, description string) *System {
	// p uses s (implied)
	s.design.addRelationship(p, s, RelImpliedUse, description)
	return s
}

// UsedBy creates a "USES" relationship from the given person to this system.
func (s *System) UsedBy(p *Person, description string) *System {
	s.design.addRelationship(p, s, RelUses, description)
	return s
}

func (s *System) Uses(n INode, description string) *System {
	s.design.addRelationship(s, n, RelUses, description)
	return s
}

// Tag adds a tag (chainable).
func (s *System) Tag(t string) *System {
	s.Node.Tag(t)
	return s
}

func (s *System) External() *System {
	s.Node.External()
	return s
}

// Container creates a new Container and (by convention) relates the system->container
// with "BELONGS_TO". You can adapt as needed.
func (s *System) Container(name, description string) *Container {
	container := &Container{
		Node:   NewNodeWithParent(s, s.design, name, description, NodeTypeContainer),
		system: s,
	}
	s.design.nodes[container.Node.ID] = container.Node

	// We record that the container belongs to this system
	s.design.addRelationship(container, s, RelBelongsTo, "Is part of")

	return container
}

// -----------------------------------------------------------------------------

// Container represents a "Container" node in C4.
type Container struct {
	*Node
	system *System // Link back to the parent system
}

// UsedBy creates a "USES" relationship from the given person to this container.
func (c *Container) UsedBy(p INode, description string) *Container {
	// p uses c: add explicit relationship: p -> container
	c.design.addRelationship(p, c, RelUses, description)
	// Propagate: p also uses container's system: add implied relationship p -> system
	c.system.ImpliedUsedBy(p, description)
	return c
}

func (c *Container) Uses(n INode, description string) *Container {
	// c uses n: add explicit relationship: container -> target
	c.design.addRelationship(c, n, RelUses, description)

	// c.system impled usage: c.system.system -> n
	c.system.ImpliedUse(n, description)

	// If the target node belongs to a container, create an implied relationship: using system -> target container's system
	if targetContainer, ok := n.(*Container); ok && targetContainer.system != nil {
		if targetContainer.system.ID != c.system.ID {
			c.system.ImpliedUse(targetContainer.system, description+" (implied "+targetContainer.ID+"--"+c.system.ID+")")
		}
	}

	// If the target node belongs to a component, create an implied relationship: using system -> target component's system
	if targetComponent, ok := n.(*Component); ok && targetComponent.container != nil && targetComponent.container.system != nil {
		if targetComponent.container.system.ID != c.system.ID {
			c.system.ImpliedUse(targetComponent.container.system, description)
		}
	}

	return c
}

func (c *Container) ImpliedUsedBy(p INode, description string) *Container {
	// p uses c (implied)
	c.design.addRelationship(p, c, RelImpliedUse, description)
	// Propagate: p also uses c.system (implied) as p -> system
	c.system.ImpliedUsedBy(p, description)
	return c
}

// Component creates a new Component and relates container->component with BELONGS_TO.
func (c *Container) Component(name, description string) *Component {
	return c.ComponentWithId(name, name, description)
}

func (c *Container) ComponentWithId(id string, name, description string) *Component {
	component := &Component{
		Node:      NewNodeWithIdAndParent(id, c, c.design, name, description, NodeTypeComponent),
		container: c,
	}
	c.design.nodes[component.Node.ID] = component.Node

	// We record that the component belongs to this container
	c.design.addRelationship(component, c, RelBelongsTo, "Is part of")

	return component
}

func (c *Container) Custom(label string, name string, description string, belongsToDescription ...string) *CustomComponent {
	component := &CustomComponent{
		Node:      NewNodeWithIdAndParent(name, c, c.design, name, description, NodeType(label)),
		container: c,
	}
	c.design.nodes[component.Node.ID] = component.Node

	// We record that the component belongs to this container
	finalBelongsToDescription := "Belongs to"
	if len(belongsToDescription) > 0 {
		finalBelongsToDescription = belongsToDescription[0]
	}
	c.design.addRelationship(component, c, RelBelongsTo, finalBelongsToDescription)

	return component
}

type CustomComponent struct {
	*Node
	container *Container
}

func (c *CustomComponent) Custom(label string, name string, description string, belongsToDescription ...string) *CustomComponent {
	return c.CustomWithId(name, label, name, description, belongsToDescription...)
}

func (c *CustomComponent) CustomWithId(id string, label string, name string, description string, belongsToDescription ...string) *CustomComponent {
	component := &CustomComponent{
		Node:      NewNodeWithIdAndParent(id, c, c.design, name, description, NodeType(label)),
		container: c.container,
	}
	c.design.nodes[component.Node.ID] = component.Node

	// We record that the component belongs to this container
	finalBelongsToDescription := "Belongs to"
	if len(belongsToDescription) > 0 {
		finalBelongsToDescription = belongsToDescription[0]
	}
	c.design.addRelationship(component, c, RelBelongsTo, finalBelongsToDescription)

	return component
}

func (c *CustomComponent) Tag(tag string) *CustomComponent {
	c.Node.Tag(tag)
	return c
}

func (c *CustomComponent) Uses(n INode, description string) *CustomComponent {
	c.design.addRelationship(c, n, RelUses, description)
	return c
}

func (c *CustomComponent) UsedBy(p INode, description string) *CustomComponent {
	c.design.addRelationship(p, c, RelUses, description)
	c.container.ImpliedUsedBy(p, description) // Also relate container->person
	return c
}

// -----------------------------------------------------------------------------

// Component represents a "Component" node in C4.
type Component struct {
	*Node
	container *Container
}

func (c *Component) AddLabel(label string) *Component {
	c.Node.AddLabel(label)
	return c
}

func (c *Component) Custom(label string, name string, description string, belongsToDescription ...string) *CustomComponent {
	component := &CustomComponent{
		Node:      NewNodeWithIdAndParent(name, c, c.design, name, description, NodeType(label)),
		container: c.container,
	}
	c.design.nodes[component.Node.ID] = component.Node

	// We record that the component belongs to this container
	finalBelongsToDescription := "Belongs to"
	if len(belongsToDescription) > 0 {
		finalBelongsToDescription = belongsToDescription[0]
	}
	c.design.addRelationship(component, c, RelBelongsTo, finalBelongsToDescription)

	return component
}

func (c *Component) Uses(n INode, description string) *Component {
	c.design.addRelationship(c, n, RelUses, description)

	// If the target node belongs to a container, create an implied relationship:
	// component's system uses target container's system (c.container.system -> targetContainer.system)
	if targetContainer, ok := n.(*Container); ok && targetContainer.system != nil {
		c.container.system.ImpliedUsedBy(targetContainer.system, description)
	}

	// If the target node belongs to a component, create an implied relationship:
	// component's system uses target component's system (c.container.system -> targetComponent.container.system)
	if targetComponent, ok := n.(*Component); ok && targetComponent.container != nil && targetComponent.container.system != nil {
		c.container.system.ImpliedUsedBy(targetComponent.container.system, description)
	}

	return c
}

// UsedBy creates a "USES" relationship from the given person to this component.
func (c *Component) UsedBy(p INode, description string) *Component {
	c.design.addRelationship(p, c, RelUses, description)
	c.container.ImpliedUsedBy(p, description) // Also relate container->person
	return c
}

// -----------------------------------------------------------------------------
// Design: the container for all nodes & relationships
// -----------------------------------------------------------------------------

// Design represents a C4 model
type Design struct {
	ID                string
	Name              string
	Description       string
	nodes             map[string]*Node
	relationships     []Relationship
	impliedUseEnabled bool
}

// NewDesign creates a new C4 design
func NewDesign(name, description string) *Design {
	d := &Design{
		ID:          "design_" + name,
		Name:        name,
		Description: description,
		nodes:       map[string]*Node{},
	}
	d.nodes[d.ID] = &Node{
		ID:          d.ID,
		Name:        name,
		Description: description,
		NodeType:    NodeTypeDesign,
		Tags:        []string{"design"},
		IsExternal:  false,
		design:      d,
	}
	return d

}

// NodeReference fetches an element from the design by its ID.
func (d *Design) NodeReference(id string) INode {
	node, ok := d.nodes[id]
	if !ok {
		d.nodes[id] = &Node{
			ID:          id,
			NodeType:    NodeTypeUnknown,
			Name:        id,
			Description: "Unknown node",
		}
		node = d.nodes[id]
	}
	return node
}

type NodeReference struct {
	ID           string
	resolvedNode INode // populated when resolved
}

func (n *NodeReference) FullId() string {
	return n.ID
}

func (n *NodeReference) FullName() string {
	return n.resolvedNode.FullName()
}

func (n *NodeReference) GetID() string {
	return n.ID
}

// EnableImpliedUse enables or disables implied use relationships.
// Implied use relationships are relationships that are not explicitly created but are implied by the relationships between nodes.
func (d *Design) EnableImpliedUse(enable bool) {
	d.impliedUseEnabled = enable
}

// Person constructs a Person node in this Design.
func (d *Design) Person(name, description string) *Person {
	//  NewNodeWithIdAndParent(name, nil, nil, name, description, nodeType)
	p := &Person{
		Node:   NewNodeWithIdAndParent("person_"+name, d, d, name, description, NodeTypePerson),
		design: d,
	}
	p.Node.design = d // Set design reference
	d.addRelationship(p, d, RelBelongsTo, "Belongs to")

	d.nodes[p.Node.ID] = p.Node
	return p
}

func (d *Design) GetID() string {
	return d.ID
}

func (d *Design) FullName() string {
	return d.Name
}

func (d *Design) FullNameSlice() []string {
	return []string{d.Name}
}

func (d *Design) FullId() string {
	return d.ID
}

// System constructs a System node in this Design.
func (d *Design) System(name, description string) *System {
	return d.SystemWithId(name, name, description)
}

func (d *Design) SystemWithId(id string, name, description string) *System {
	s := &System{
		Node:   NewNodeWithIdAndParent(id, d, d, name, description, NodeTypeSystem),
		design: d,
	}
	s.Node.design = d // Set design reference
	d.addRelationship(s, d, RelBelongsTo, "Belongs to")
	d.nodes[s.Node.ID] = s.Node
	return s
}

// addRelationship is a helper to record relationships in the design.
// It takes start and end nodes, relationship type, and a description.
func (d *Design) addRelationship(startNode, endNode INode, relType RelationshipType, desc string) {
	if !d.impliedUseEnabled && relType == RelImpliedUse {
		return
	}

	// check if endId belongs_to startId, and we're adding a implied use from startId to endId, ignore
	if relType == RelImpliedUse {
		for _, rel := range d.relationships {
			if rel.StartID == endNode.FullId() && rel.EndID == startNode.FullId() && rel.Type == RelBelongsTo {
				return
			}
		}
	}

	d.relationships = append(d.relationships, Relationship{
		StartID:     startNode.FullId(),
		EndID:       endNode.FullId(),
		Type:        relType,
		Description: desc,
	})
}

// DeleteFromNeo4j removes the design and all its related nodes and relationships from the Neo4j database.
func DeleteFromNeo4j(ctx context.Context, designId string, driver neo4j.DriverWithContext) error {
	session := driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// Match the Design node and delete it along with all related nodes and relationships
		query := `
MATCH (design :Design{id: $designID})-[r*0..]->(related)
DETACH DELETE design, related
`
		_, e := tx.Run(ctx, query, map[string]any{"designID": designId})
		return nil, e
	})
	return err
}

// DeleteFromNeo4j removes the design and all its related nodes and relationships from the Neo4j database.
func (d *Design) DeleteFromNeo4j(ctx context.Context, driver neo4j.DriverWithContext) error {
	return DeleteFromNeo4j(ctx, d.ID, driver)
}

// SaveToNeo4j pushes the entire model to the Neo4j database
func (d *Design) SaveToNeo4j(ctx context.Context, driver neo4j.DriverWithContext, sessConfig neo4j.SessionConfig) error {
	session := driver.NewSession(ctx, sessConfig)
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// MERGE all nodes
		for _, node := range d.nodes {
			setStr := "n.name=$name, n.description=$desc, n.nodeType=$nodeType, n.tags=$tags"
			params := map[string]any{
				"id":       node.FullId(),
				"name":     node.Name,
				"desc":     node.Description,
				"nodeType": string(node.NodeType),
				"tags":     node.Tags,
			}
			for _, tag := range node.Tags {
				tag = strings.ReplaceAll(tag, `-`, `_`)
				tag = strings.ReplaceAll(tag, `:`, `_`)
				tag = strings.ReplaceAll(tag, ` `, `_`)
				tag = strings.ReplaceAll(tag, `"`, `_`)
				tag = strings.ReplaceAll(tag, `'`, `_`)
				setStr += ", n.tag_" + tag + "=$tag_" + tag
				params["tag_"+tag] = tag
			}
			if node.IsExternal {
				setStr += ", n.external=$ext"
				params["ext"] = node.IsExternal
			}

			query := strings.Builder{}

			if len(node.Labels) > 0 {
				query.WriteString(`MERGE (n:` + string(node.NodeType))
				for _, label := range node.Labels {
					query.WriteString(`:` + label)
				}
				query.WriteString(` { id: $id })`)
			} else {
				query.WriteString(`MERGE (n:` + string(node.NodeType) + ` { id: $id })`)
			}
			query.WriteString(`
ON CREATE SET ` + setStr + `
ON MATCH SET  ` + setStr + `
`)

			if _, e := tx.Run(ctx, query.String(), params); e != nil {
				return nil, e
			}
		}

		// MERGE all relationships
		for _, rel := range d.relationships {
			startNodeLabel := "Unknown"
			endNodeLabel := "Unknown"
			for _, node := range d.nodes {
				if node.FullId() == rel.StartID {
					startNodeLabel = string(node.NodeType)
				}
				if node.FullId() == rel.EndID {
					endNodeLabel = string(node.NodeType)
				}
			}
			query := fmt.Sprintf(`
MERGE (start:%s { id: $startID })
MERGE (end:%s { id: $endID })
MERGE (start)-[r:%s { description: $desc }]->(end)
`, startNodeLabel, endNodeLabel, rel.Type)

			params := map[string]any{
				"startID": rel.StartID,
				"endID":   rel.EndID,
				"desc":    rel.Description,
			}
			if tmp, e := tx.Run(ctx, query, params); e != nil {
				return nil, e
			} else {
				if _, e := tmp.Consume(ctx); e != nil {
					return nil, e
				} else {
					// fmt.Println("Relationship created:", res)
				}
			}
		}
		return nil, nil
	})
	return err
}

// ClearNeo4j_UNSAFE deletes all nodes and relationships in the Neo4j database.
func ClearNeo4j_UNSAFE(ctx context.Context, driver neo4j.DriverWithContext, sessConfig neo4j.SessionConfig) error {
	session := driver.NewSession(ctx, sessConfig)
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
MATCH (n)
DETACH DELETE n
`
		_, e := tx.Run(ctx, query, nil)
		return nil, e
	})
	return err
}

// MD5 returns the MD5 hash of a string.
func MD5(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
