package main

import (
	"context"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	. "github.com/wricardo/neoarch"
)

func main() {
	// Create a new design
	design := NewDesign("Example 1", "Something")

	// Create Persons
	user := design.Person("User", "Any internet user").
		External().
		Tag("person")

	developer := design.Person("Developer", "developer/employee").
		Tag("person").
		Tag("developer")

	// Person -> Person usage
	user.InteractsWith(developer, "Interacts with developer")

	// Create a System with nested containers/components
	someSystem := design.System("SomeSystem", "API system").
		// UsedBy(user, "Uses the system").
		Tag("system")

	api := someSystem.Container("API", "Backend API for Doxy").
		// UsedBy(user, "Uses the system").
		Tag("system")

	api.Component("GraphQL", "GraphQL API").
		UsedBy(user, "Uses the API").
		Tag("component").
		Tag("graphql")

	// Connect to Neo4j and save everything
	neo4jURI := "neo4j://localhost:7687" // Example connection URI
	username := "neo4j"
	password := "neo4jneo4j"

	driver, err := neo4j.NewDriverWithContext(neo4jURI, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatal("Failed to create driver:", err)
	}

	defer driver.Close(context.Background())
	clearNeo4j(driver)

	// Push the design to Neo4j
	if err := design.SaveToNeo4j(context.Background(), driver); err != nil {
		log.Fatalf("Failed to save design: %v", err)
	} else {
		fmt.Println("Design saved successfully to Neo4j!")
	}
}

// clearNeo4j is a helper to wipe all nodes & relationships in the DB.
func clearNeo4j(driver neo4j.DriverWithContext) {
	session := driver.NewSession(context.Background(), neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(context.Background())

	_, err := session.Run(context.Background(), "MATCH (n) DETACH DELETE n", nil)
	if err != nil {
		log.Fatalf("Failed to clear Neo4j: %v", err)
	} else {
		fmt.Println("Neo4j cleared successfully!")
	}
}
