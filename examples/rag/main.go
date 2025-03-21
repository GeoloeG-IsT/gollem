package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
	"github.com/GeoloeG-IsT/gollem/pkg/rag"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}

	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("Warning: OPENAI_API_KEY not found in environment, using dummy key")
		apiKey = "dummy_openai_api_key_12345"
	}

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gollem-rag-example")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test documents
	createTestDocuments(tempDir)

	// Create an LLM provider
	provider, err := openai.NewProvider(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4",
	})
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create an embedding provider
	embeddings := rag.NewEmbeddings(provider, "text-embedding-ada-002", 1536)

	// Create a RAG system
	system := rag.NewRAGSystem(provider, embeddings)

	// Add documents
	ctx := context.Background()
	err = system.AddDirectory(ctx, tempDir)
	if err != nil {
		log.Fatalf("Failed to add directory: %v", err)
	}

	fmt.Println("Documents added to RAG system.")

	// Set query options
	system.SetQueryOptions(rag.QueryOptions{
		NumDocuments:   3,
		IncludeMetadata: true,
		PromptTemplate: "Answer the question based on the following context:\n\n{{context}}\n\nQuestion: {{query}}",
	})

	// Query the system
	fmt.Println("\nQuerying: What is the capital of France?")
	response, err := system.Query(ctx, "What is the capital of France?")
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	// Print the response
	fmt.Println("Response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)

	// Query the system again
	fmt.Println("\nQuerying: What is the capital of Germany?")
	response, err = system.Query(ctx, "What is the capital of Germany?")
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	// Print the response
	fmt.Println("Response:", response.Text)
	fmt.Println("Tokens used:", response.TokensUsed.Total)
}

// createTestDocuments creates test documents in the specified directory
func createTestDocuments(dir string) {
	// Create a document about France
	franceDoc := `France is a country in Western Europe. Its capital is Paris, which is known as the City of Light.
Paris is famous for landmarks such as the Eiffel Tower, the Louvre Museum, and Notre-Dame Cathedral.
France has a population of about 67 million people and is known for its cuisine, wine, and culture.`

	// Create a document about Germany
	germanyDoc := `Germany is a country in Central Europe. Its capital is Berlin, which is known for its history and culture.
Berlin is famous for landmarks such as the Brandenburg Gate, the Berlin Wall, and the Reichstag building.
Germany has a population of about 83 million people and is known for its engineering, beer, and festivals.`

	// Create a document about Italy
	italyDoc := `Italy is a country in Southern Europe. Its capital is Rome, which is known as the Eternal City.
Rome is famous for landmarks such as the Colosseum, the Vatican, and the Trevi Fountain.
Italy has a population of about 60 million people and is known for its art, food, and fashion.`

	// Write the documents to files
	err := os.WriteFile(filepath.Join(dir, "france.txt"), []byte(franceDoc), 0644)
	if err != nil {
		log.Fatalf("Failed to write France document: %v", err)
	}

	err = os.WriteFile(filepath.Join(dir, "germany.txt"), []byte(germanyDoc), 0644)
	if err != nil {
		log.Fatalf("Failed to write Germany document: %v", err)
	}

	err = os.WriteFile(filepath.Join(dir, "italy.txt"), []byte(italyDoc), 0644)
	if err != nil {
		log.Fatalf("Failed to write Italy document: %v", err)
	}

	fmt.Println("Created test documents in", dir)
}
