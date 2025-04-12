// Package neoarch provides a domain-specific language for modeling C4 architecture
// and storing it in a Neo4j database. It allows creating and connecting architecture
// elements such as Persons, Systems, Containers, and Components, following the
// C4 model principles (https://c4model.com).
package neoarch

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// -----------------------------------------------------------------------------
// Enumerations & Basic Types
// -----------------------------------------------------------------------------

// NodeType is an enum-like type for different C4 elements.
type NodeType string

const (
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
}

// -----------------------------------------------------------------------------
// Core Node & Derived Types
// -----------------------------------------------------------------------------

// Node is the shared struct for all C4 elements.
type Node struct {
	ID          string   // Unique identifier (could be the "name")
	Name        string   // Display name
	Description string   // Brief description
	NodeType    NodeType // e.g. Person, System, Container, Component
	Tags        []string // Arbitrary tags
	IsExternal  bool     // For marking external nodes
	design      *Design  // Link back to the containing Design
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

// Person can also use “UsedBy” if you want to invert direction, but here we
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
		Node: &Node{
			ID:          fmt.Sprintf("%s.cr_%s", s.ID, name), // For uniqueness
			Name:        name,
			Description: description,
			NodeType:    NodeTypeContainer,
			design:      s.design,
		},
		system: s,
	}
	s.design.nodes = append(s.design.nodes, container.Node)

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

	// If the target node belongs to a container, create an implied relationship: using system -> target container’s system
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
	component := &Component{
		container: c,
		Node: &Node{
			ID:          fmt.Sprintf("%s.ct_%s", c.ID, name),
			Name:        name,
			Description: description,
			NodeType:    NodeTypeComponent,
			design:      c.design,
		},
	}
	c.design.nodes = append(c.design.nodes, component.Node)

	// We record that the component belongs to this container
	c.design.addRelationship(component, c, RelBelongsTo, "Container contains Component")

	return component
}

// -----------------------------------------------------------------------------

// Component represents a "Component" node in C4.
type Component struct {
	*Node
	container *Container
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
	ID            string
	Name          string
	Description   string
	nodes         []*Node
	relationships []Relationship
}

// NewDesign creates a new C4 design
func NewDesign(name, description string) *Design {
	return &Design{
		ID:          "design_" + name,
		Name:        name,
		Description: description,
		nodes:       []*Node{},
	}
}

// Person constructs a Person node in this Design.
func (d *Design) Person(name, description string) *Person {
	p := &Person{
		Node: &Node{
			ID:          "person_" + name,
			Name:        name,
			Description: description,
			NodeType:    NodeTypePerson,
			design:      d,
		},
		design: d,
	}
	d.nodes = append(d.nodes, p.Node)
	return p
}

// System constructs a System node in this Design.
func (d *Design) System(name, description string) *System {
	s := &System{
		Node: &Node{
			ID:          d.ID + ".ss_" + name,
			Name:        name,
			Description: description,
			NodeType:    NodeTypeSystem,
			design:      d,
		},
		design: d,
	}
	d.nodes = append(d.nodes, s.Node)
	return s
}

// addRelationship is a helper to record relationships in the design.
// It takes start and end nodes, relationship type, and a description.
func (d *Design) addRelationship(startID, endID INode, relType RelationshipType, desc string) {
	// check if endId belongs_to startId, and we're adding a implied use from startId to endId, ignore
	if relType == RelImpliedUse {
		for _, rel := range d.relationships {
			if rel.StartID == endID.GetID() && rel.EndID == startID.GetID() && rel.Type == RelBelongsTo {
				return
			}
		}
	}

	d.relationships = append(d.relationships, Relationship{
		StartID:     startID.GetID(),
		EndID:       endID.GetID(),
		Type:        relType,
		Description: desc,
	})
}

// SaveToNeo4j pushes the entire model to the Neo4j database
func (d *Design) SaveToNeo4j(driver neo4j.Driver) error {
	session := driver.NewSession(neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close()

	_, err := session.WriteTransaction(func(tx neo4j.Transaction) (any, error) {
		// MERGE all nodes
		for _, node := range d.nodes {
			setStr := "n.name=$name, n.description=$desc, n.nodeType=$nodeType, n.tags=$tags"
			params := map[string]any{
				"id":       node.ID,
				"name":     node.Name,
				"desc":     node.Description,
				"nodeType": string(node.NodeType),
				"tags":     node.Tags,
			}
			for _, tag := range node.Tags {
				setStr += ", n.tag_" + tag + "=$tag_" + tag
				params["tag_"+tag] = tag
			}
			if node.IsExternal {
				setStr += ", n.external=$ext"
				params["ext"] = node.IsExternal
			}

			query := `
MERGE (n:` + string(node.NodeType) + ` { id: $id })
ON CREATE SET ` + setStr + `
ON MATCH SET  ` + setStr + `
`
			if _, e := tx.Run(query, params); e != nil {
				return nil, e
			}
		}

		// MERGE all relationships
		for _, rel := range d.relationships {
			query := fmt.Sprintf(`
MATCH (start { id: $startID })
MATCH (end { id: $endID })
MERGE (start)-[r:%s { description: $desc }]->(end)
`, rel.Type)

			params := map[string]any{
				"startID": rel.StartID,
				"endID":   rel.EndID,
				"desc":    rel.Description,
			}
			if _, e := tx.Run(query, params); e != nil {
				return nil, e
			}
		}
		return nil, nil
	})
	return err
}
