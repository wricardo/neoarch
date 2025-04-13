# 🧠 neoarch: C4 Modeling DSL for Go + Neo4j

**neoarch** is a Go library that provides a Domain Specific Language (DSL) for constructing and persisting C4 architecture models using Go + Neo4j. With neoarch, you can define various architectural elements—**Persons**, **Systems**, **Containers**, and **Components**—and establish meaningful relationships between them using Go programing language through intuitive and chainable methods. It lets you create, query and visualize your system structure through graph databases.

---
<img width="2000" alt="Screenshot 2025-04-13 at 4 49 24 AM" src="https://github.com/user-attachments/assets/efe1809d-6796-4667-a6c0-29337ef19154" />


## 📚 Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
  - [Creating a Design](#creating-a-design)
  - [Defining Elements](#defining-elements)
  - [Persisting to Neo4j](#persisting-to-neo4j)
- [Example](#example)
- [Getting Started](#getting-started)
- [Contribute](#contribute)
- [License](#license)

---

## ✨ Features

- **Intuitive DSL:** Define architectural elements like Persons, Systems, Containers, and Components with expressive, fluent syntax for building hierarchical system maps.
- **Relationship Modeling:** Easily establish meaningful connections such as `InteractsWith`, `Uses`, and `BelongsTo`, reflecting communication and ownership.
- **Neo4j Integration:** Persist your C4 models directly into a Neo4j graph database for powerful visualization, impact analysis, and system evolution tracking.
- **Programmatic Design:** Create and modify your architecture models programmatically, allowing for dynamic generation based on resources read with SDKS (aws-sdk-go, etc).



---

## 🛠 Installation

1. **Install Go:**  
   Make sure you have [Go](https://golang.org/dl/) installed (Go 1.24.2 or later is recommended).

2. **Initialize your project and add neoarch:**

   ```bash
   mkdir my-c4-project
   cd my-c4-project
   go mod init github.com/yourusername/my-c4-project
   go get github.com/wricardo/neoarch
   ```

3. **Neo4j Go Driver:**  
   The Neo4j Go Driver is managed automatically via your module file. No extra steps required.

4. **Neo4j Setup:**
   Make sure you have a running Neo4j instance (cloud or local). Default config assumes `neo4j://localhost:7687`.

---

## 🚀 Usage

### 🧱 Creating a Design

Start by initializing a new C4 model design object, which acts as a container for all elements and relationships.

```go
import "github.com/wricardo/neoarch"

design := neoarch.NewDesign("My Architecture", "An example C4 model")
```

### 🧩 Defining Elements

Use fluent methods to define people, systems, containers, and components, as well as how they interact.

```go
// Create a Person node
user := design.Person("User", "A generic internet user").
    External().
    Tag("person")

// Create another Person
developer := design.Person("Developer", "A developer or employee").
    Tag("developer")

// Create a relationship
user.InteractsWith(developer, "Interacts with the developer")

// Define a System node
system := design.System("API System", "A backend API system").
    Tag("system")

// Add a Container inside the System
apiContainer := system.Container("API", "Backend API container").
    Tag("container")

// Add a Component inside the Container
apiContainer.Component("GraphQL", "GraphQL API service").
    UsedBy(user, "User uses the GraphQL API")
```

> You can define as many elements and relationships as you need. Each element keeps track of its links to others, which helps build accurate dependency maps.

### 🔄 Persisting to Neo4j

To save your architecture design to a Neo4j database, follow this pattern:

```go
import (
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
    "log"
)

neo4jURI := "neo4j://localhost:7687"
username := "neo4j"
password := "your-password"

driver, err := neo4j.NewDriver(neo4jURI, neo4j.BasicAuth(username, password, ""))
if err != nil {
    log.Fatal("Failed to create driver:", err)
}
defer driver.Close()

// Optional: Clear existing data before persisting
clearNeo4j(driver)

// Save the design
if err := design.SaveToNeo4j(driver); err != nil {
    log.Fatalf("Failed to save design: %v", err)
}
```

> ⚠️ **Note:** The helper function `clearNeo4j(driver)` removes all existing data. Use cautiously in production.

---

## 🧪 Example

A complete working example is available in the `examples/example1` directory. This includes setup, model creation, and persistence.

### ▶️ Run the Example

```bash
cd examples/example1

go run main.go
```

The example demonstrates:
- How to define an architecture programmatically
- How to build relationships visually represented in Neo4j
- The power of mapping your software in C4 using Go

For advanced examples or enterprise scenarios, you can create multiple systems and connect them with cross-domain references.

---

## ⚡ Getting Started

### 📥 Clone the Repository

```bash
git clone https://github.com/wricardo/neoarch.git
cd neoarch
```

### 📦 Install Dependencies

```bash
go mod download
```

### ✅ Run Examples or Tests

Explore the provided examples under `examples/` to get familiar with the library.
Create custom tests or integrate into your CI/CD workflows to validate model consistency.

> Tip: You can create tooling on top of neoarch to export diagrams, check compliance, or integrate with internal developer portals.

---

## 🤝 Contribute

Contributions, bug reports, and feature suggestions are very welcome!

- 💬 Open an issue to discuss bugs or new features
- 🌱 Fork and submit a pull request for enhancements
- 🧪 Improve test coverage or extend DSL capabilities

We aim to keep the API stable and well-documented. Help us improve by contributing your use cases.

---

## 📄 License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for full details.

---

Built with ❤️ by [@wricardo](https://github.com/wricardo) — Let’s model the future of software architecture!

