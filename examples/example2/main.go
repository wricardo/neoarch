package main

import (
	"context"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/wricardo/neoarch"
)

func main() {
	design := neoarch.NewDesign("Twitter Clone", "Social + User Systems with API Gateway, GraphQL, gRPC")

	// ------------------------------
	// USER SYSTEM
	// ------------------------------
	userSystem := design.System("UserSystem", "Handles user management and authentication")

	userAPIGateway := userSystem.Container("User API Gateway", "HTTP entrypoint").
		Tag("gateway")
	userGraphQL := userSystem.Container("User GraphQL", "Orchestrates queries and mutations").
		Tag("graphql").
		Uses(userAPIGateway, "Receives traffic from")
	userDB := userSystem.Container("User DB", "Stores user info").
		Tag("db")
	userS3 := userSystem.Container("User S3", "Stores avatars").
		Tag("s3")
	userTemporal := userSystem.Container("User Temporal Worker", "Handles background workflows").
		Tag("temporal")

	userService := userSystem.Container("User gRPC Service", "Handles core user operations").
		Tag("grpc").
		Uses(userDB, "Reads/writes user data").
		Uses(userS3, "Stores profile images").
		Uses(userTemporal, "Schedules background jobs")

	// Cross-layer
	userGraphQL.Uses(userService, "Resolves user operations")

	// GraphQL Components
	userGraphQL.Component("Schema Definition", "Defines User types and fields")
	userGraphQL.Component("Query Resolver", "Handles fetching user data")
	userGraphQL.Component("Mutation Resolver", "Handles signup, update, etc.")
	userGraphQL.Component("Middleware", "Cross-cutting GraphQL logic")
	userGraphQL.Component("Authorization", "Enforces auth rules")

	// gRPC Components
	userService.Component("Handler", "Request-level handling")
	userService.Component("Service", "Business logic")
	userService.Component("Repository", "Persistence layer")

	// ------------------------------
	// SOCIAL SYSTEM
	// ------------------------------
	socialSystem := design.System("SocialSystem", "Handles tweets, follows, feeds")

	socialAPIGateway := socialSystem.Container("Social API Gateway", "HTTP entrypoint").
		Tag("gateway")
	socialGraphQL := socialSystem.Container("Social GraphQL", "Manages tweet/feed queries").
		Tag("graphql").
		Uses(socialAPIGateway, "Receives traffic from")
	socialDB := socialSystem.Container("Social DB", "Stores tweets, follows").
		Tag("db")
	socialS3 := socialSystem.Container("Social S3", "Stores tweet media").
		Tag("s3")
	socialTemporal := socialSystem.Container("Social Temporal Worker", "Feed generation and cleanup").
		Tag("temporal")

	tweetService := socialSystem.Container("Tweet gRPC Service", "Tweet logic").
		Tag("grpc").
		Uses(socialDB, "Reads/writes tweet data").
		Uses(socialS3, "Stores media").
		Uses(socialTemporal, "Schedules tweet workflows")

	followService := socialSystem.Container("Follow gRPC Service", "Follow/unfollow logic").
		Tag("grpc").
		Uses(socialDB, "Updates following/follower lists").
		Uses(socialTemporal, "Schedules notifications")

	//Cross - layer
	socialGraphQL.Uses(tweetService, "Resolves tweet ops")
	socialGraphQL.Uses(followService, "Resolves follow ops")

	// Inter-system call
	tweetService.Uses(userService, "Fetch user profile info for tweets")
	followService.Uses(userService, "Resolve target user")

	// GraphQL Components
	socialGraphQL.Component("Schema Definition", "Defines Tweet and Feed types")
	socialGraphQL.Component("Query Resolver", "Handles fetching tweets/feed")
	socialGraphQL.Component("Mutation Resolver", "Creates tweets, follows")
	socialGraphQL.Component("Middleware", "Logging, timing, tracing")
	socialGraphQL.Component("Authorization", "Check user permissions")

	// gRPC Components
	tweetService.Component("Handler", "gRPC entrypoint")
	tweetService.Component("Service", "Tweet logic")
	tweetService.Component("Repository", "Tweet persistence")

	followService.Component("Handler", "gRPC entrypoint")
	followService.Component("Service", "Follow logic")
	followService.Component("Repository", "Follow persistence")

	// Save to Neo4j
	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext("neo4j://localhost:7687", neo4j.BasicAuth("neo4j", "neo4jneo4j", ""))
	if err != nil {
		log.Fatal(err)
	}
	defer driver.Close(ctx)

	// neoarch.ClearNeo4j_UNSAFE(driver)

	if err := design.SaveToNeo4j(ctx, driver, neo4j.SessionConfig{DatabaseName: "neo4j"}); err != nil {
		log.Fatal("Failed to persist design:", err)
	}

}
