# RAG Documentation

This document provides detailed information about the Retrieval Augmented Generation (RAG) components in Gollem.

## Overview

RAG combines retrieval-based and generation-based approaches to improve the quality and factuality of LLM responses. Gollem provides a complete set of components for building RAG applications:

1. **Document Loading**: Load documents from various sources
2. **Text Splitting**: Split documents into manageable chunks
3. **Vector Storage**: Store and retrieve document embeddings
4. **Embedding Generation**: Generate embeddings for documents and queries
5. **Retrieval**: Find relevant documents for a query
6. **Generation**: Generate responses based on retrieved documents

## Basic Usage

Here's a simple example of using RAG:

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/providers/openai"
	"github.com/GeoloeG-IsT/gollem/pkg/rag"
)

func main() {
	// Create an LLM provider
	provider, err := openai.NewProvider(openai.Config{
		APIKey: "your-api-key",
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
	err = system.AddFile(ctx, "/path/to/document.txt")
	if err != nil {
		log.Fatalf("Failed to add file: %v", err)
	}

	// Query the system
	response, err := system.Query(ctx, "What is the capital of France?")
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}

	// Print the response
	fmt.Println(response.Text)
}
```

## Components

### Document

The `Document` type represents a document in the RAG system:

```go
type Document struct {
	// ID is a unique identifier for the document
	ID string

	// Content is the text content of the document
	Content string

	// Metadata contains additional information about the document
	Metadata map[string]interface{}
}
```

### Document Loader

Document loaders load documents from various sources:

```go
// Create a file loader
loader := rag.NewFileLoader("/path/to/documents")

// Load a single document
doc, err := loader.LoadDocument(ctx, "document.txt")

// Load multiple documents from a directory
docs, err := loader.LoadDocuments(ctx, "/path/to/documents")
```

### Text Splitter

Text splitters split documents into smaller chunks:

```go
// Create a character text splitter
splitter := rag.NewCharacterTextSplitter(
	rag.WithChunkSize(1000),
	rag.WithChunkOverlap(200),
)

// Split text
chunks := splitter.SplitText("This is a long document...")

// Split a document
splitDocs := splitter.SplitDocument(doc)
```

### Vector Store

Vector stores store and retrieve document embeddings:

```go
// Create a memory vector store
store := rag.NewMemoryVectorStore(embeddings)

// Add documents
err := store.AddDocuments(ctx, docs)

// Search for similar documents
results, err := store.SimilaritySearch(ctx, "What is the capital of France?", 3)

// Clear the store
err := store.Clear(ctx)
```

### Embedding Provider

Embedding providers generate embeddings for documents and queries:

```go
// Create an embedding provider
embeddings := rag.NewEmbeddings(provider, "text-embedding-ada-002", 1536)

// Generate an embedding for a query
queryEmbedding, err := embeddings.EmbedQuery(ctx, "What is the capital of France?")

// Generate embeddings for documents
docEmbeddings, err := embeddings.EmbedDocuments(ctx, []string{"Paris is the capital of France.", "Berlin is the capital of Germany."})
```

### RAG

The `RAG` type combines all components:

```go
// Create a RAG
ragSystem := rag.NewRAG(store, provider, splitter)

// Add documents
err := ragSystem.AddDocuments(ctx, docs)

// Query
response, err := ragSystem.Query(ctx, "What is the capital of France?", 3)
```

### Query Engine

Query engines provide more control over the RAG process:

```go
// Create a query engine
engine := rag.NewQueryEngine(ragSystem, rag.QueryOptions{
	NumDocuments:   3,
	IncludeMetadata: true,
	PromptTemplate: "Answer based on: {{context}}\nQuestion: {{query}}",
})

// Query
response, err := engine.Query(ctx, "What is the capital of France?")
```

## Advanced Usage

### Custom Document Loaders

You can implement custom document loaders by implementing the `DocumentLoader` interface:

```go
type DocumentLoader interface {
	// LoadDocument loads a document from a source
	LoadDocument(ctx context.Context, source string) (*Document, error)

	// LoadDocuments loads multiple documents from a source
	LoadDocuments(ctx context.Context, source string) ([]*Document, error)
}
```

### Custom Text Splitters

You can implement custom text splitters by implementing the `TextSplitter` interface:

```go
type TextSplitter interface {
	// SplitText splits text into chunks
	SplitText(text string, options ...SplitOption) []string

	// SplitDocument splits a document into multiple documents
	SplitDocument(doc *Document, options ...SplitOption) []*Document
}
```

### Custom Vector Stores

You can implement custom vector stores by implementing the `VectorStore` interface:

```go
type VectorStore interface {
	// AddDocuments adds documents to the store
	AddDocuments(ctx context.Context, docs []*Document) error

	// SimilaritySearch searches for similar documents
	SimilaritySearch(ctx context.Context, query string, k int) ([]*Document, error)

	// Clear removes all documents from the store
	Clear(ctx context.Context) error
}
```

### Custom Embedding Providers

You can implement custom embedding providers by implementing the `EmbeddingProvider` interface:

```go
type EmbeddingProvider interface {
	// EmbedQuery generates an embedding for a query
	EmbedQuery(ctx context.Context, text string) ([]float32, error)

	// EmbedDocuments generates embeddings for documents
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}
```

## RAG Pipeline

The `RAGPipeline` type provides a convenient way to process documents:

```go
// Create a pipeline
pipeline := rag.NewRAGPipeline(loader, splitter, store)

// Process a file
err := pipeline.ProcessFile(ctx, "/path/to/document.txt")

// Process a directory
err := pipeline.ProcessDirectory(ctx, "/path/to/documents")
```

## RAG System

The `RAGSystem` type provides a high-level interface for RAG:

```go
// Create a RAG system
system := rag.NewRAGSystem(provider, embeddings)

// Add a file
err := system.AddFile(ctx, "/path/to/document.txt")

// Add a directory
err := system.AddDirectory(ctx, "/path/to/documents")

// Query
response, err := system.Query(ctx, "What is the capital of France?")

// Set query options
system.SetQueryOptions(rag.QueryOptions{
	NumDocuments:   5,
	IncludeMetadata: true,
})
```

## Configuration

RAG can be configured via the configuration system:

```json
{
  "rag": {
    "enabled": true,
    "embedding_provider": "openai",
    "embedding_model": "text-embedding-ada-002",
    "chunk_size": 1000,
    "chunk_overlap": 200,
    "num_documents": 3,
    "include_metadata": true
  }
}
```

Environment variables can be used to override configuration values:

- `GOLLEM_RAG_ENABLED`: Enable or disable RAG
- `GOLLEM_RAG_EMBEDDING_PROVIDER`: Embedding provider name
- `GOLLEM_RAG_EMBEDDING_MODEL`: Embedding model name
- `GOLLEM_RAG_CHUNK_SIZE`: Chunk size for text splitting
- `GOLLEM_RAG_CHUNK_OVERLAP`: Chunk overlap for text splitting
- `GOLLEM_RAG_NUM_DOCUMENTS`: Number of documents to retrieve
- `GOLLEM_RAG_INCLUDE_METADATA`: Whether to include metadata in prompts

## Best Practices

1. **Document Preparation**: Clean and preprocess your documents before adding them to the RAG system.
2. **Chunk Size**: Choose an appropriate chunk size based on your documents and the context window of your LLM.
3. **Number of Documents**: Retrieve enough documents to provide context, but not so many that the prompt becomes too long.
4. **Prompt Templates**: Customize prompt templates to guide the LLM in using the retrieved context effectively.
5. **Metadata**: Include relevant metadata to help the LLM understand the source and context of retrieved documents.
6. **Evaluation**: Regularly evaluate the quality of RAG responses and adjust parameters as needed.

## Examples

See the [examples directory](../examples/rag) for complete RAG examples.
