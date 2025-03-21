package rag_test

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
	"github.com/GeoloeG-IsT/gollem/pkg/rag"
)

// TestDocumentLoader tests the document loader functionality
func TestDocumentLoader(t *testing.T) {
	// Create a temporary file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")

	// Write content to the file
	content := "This is a test document for RAG testing."
	err := ioutil.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Create a file loader
	loader := rag.NewFileLoader(tempDir)

	// Load the document
	ctx := context.Background()
	doc, err := loader.LoadDocument(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Failed to load document: %v", err)
	}

	// Check the document
	if doc.ID != "test.txt" {
		t.Fatalf("Document ID is incorrect: %s", doc.ID)
	}

	if doc.Content != content {
		t.Fatalf("Document content is incorrect: %s", doc.Content)
	}

	if doc.Metadata["type"] != "file" {
		t.Fatalf("Document metadata type is incorrect: %s", doc.Metadata["type"])
	}

	// Test loading multiple documents
	// Create another file
	anotherFile := filepath.Join(tempDir, "another.txt")
	err = ioutil.WriteFile(anotherFile, []byte("Another test document."), 0644)
	if err != nil {
		t.Fatalf("Failed to write another test file: %v", err)
	}

	// Load documents from the directory
	docs, err := loader.LoadDocuments(ctx, tempDir)
	if err != nil {
		t.Fatalf("Failed to load documents: %v", err)
	}

	// Check the documents
	if len(docs) != 2 {
		t.Fatalf("Loaded %d documents, expected 2", len(docs))
	}
}

// TestTextSplitter tests the text splitter functionality
func TestTextSplitter(t *testing.T) {
	// Create a text splitter
	splitter := rag.NewCharacterTextSplitter()

	// Test document splitting
	text := "This is a test document that needs to be split into chunks."
	doc := &rag.Document{
		ID:      "test",
		Content: text,
		Metadata: map[string]interface{}{
			"source": "test",
		},
	}

	splitDocs := splitter.SplitDocument(doc)

	// Check the split documents
	if len(splitDocs) == 0 {
		t.Fatal("No split documents created")
	}

	// Check that each document has the correct metadata
	for _, splitDoc := range splitDocs {
		if splitDoc.Metadata["source"] != "test" {
			t.Fatalf("Split document metadata is incorrect: %v", splitDoc.Metadata)
		}
	}
}

// TestVectorStore tests the vector store functionality
func TestVectorStore(t *testing.T) {
	// Create a mock embedding provider
	embeddings := &MockEmbeddingProvider{}

	// Create a vector store
	store := rag.NewMemoryVectorStore(embeddings)

	// Create documents and convert to chunks
	ctx := context.Background()
	chunks := []*rag.Chunk{
		{
			ID:         "chunk1",
			DocumentID: "doc1",
			Content:    "This is the first document.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
		{
			ID:         "chunk2",
			DocumentID: "doc2",
			Content:    "This is the second document.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
	}

	// Generate embeddings for each chunk
	for i := range chunks {
		embedding, err := embeddings.EmbedDocument(ctx, chunks[i].Content)
		if err != nil {
			t.Fatalf("Failed to embed chunk: %v", err)
		}
		chunks[i].Embedding = embedding
	}

	// Add chunks to the store
	err := store.AddChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("Failed to add chunks: %v", err)
	}

	// Generate embedding for the query
	queryEmbedding, err := embeddings.EmbedQuery(ctx, "first document")
	if err != nil {
		t.Fatalf("Failed to embed query: %v", err)
	}

	// Search for similar documents
	results, err := store.SimilaritySearch(ctx, queryEmbedding, 1)
	if err != nil {
		t.Fatalf("Failed to search for similar documents: %v", err)
	}

	// Check the results
	if len(results) != 1 {
		t.Fatalf("Got %d results, expected 1", len(results))
	}

	// Clear the store
	err = store.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear store: %v", err)
	}
}

// TestRAG tests the RAG functionality
func TestRAG(t *testing.T) {
	// Create a mock embedding provider
	embeddings := &MockEmbeddingProvider{}

	// Create a vector store
	store := rag.NewMemoryVectorStore(embeddings)

	// Create a mock LLM provider
	provider := &MockProvider{
		name: "mock_provider",
	}

	// Create a RAG with options
	ragSystem, err := rag.NewRAG(
		rag.WithVectorStore(store),
		rag.WithEmbeddings(embeddings),
		rag.WithChunkSize(1000),
		rag.WithChunkOverlap(200),
		rag.WithTopK(3),
	)
	if err != nil {
		t.Fatalf("Failed to create RAG: %v", err)
	}

	// Create documents
	docs := []*rag.Document{
		{
			ID:      "doc1",
			Content: "The capital of France is Paris.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
		{
			ID:      "doc2",
			Content: "The capital of Germany is Berlin.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
	}

	// Add documents to the RAG
	ctx := context.Background()
	err = ragSystem.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}

	// Query the RAG
	response, err := ragSystem.Query(ctx, "What is the capital of France?", provider)
	if err != nil {
		t.Fatalf("Failed to query RAG: %v", err)
	}

	// Check the response
	if response == nil {
		t.Fatal("Response is nil")
	}
}

// TestQueryEngine tests the query engine functionality
func TestQueryEngine(t *testing.T) {
	// Create a mock embedding provider
	embeddings := &MockEmbeddingProvider{}

	// Create a vector store
	store := rag.NewMemoryVectorStore(embeddings)

	// Create a RAG with options
	ragSystem, err := rag.NewRAG(
		rag.WithVectorStore(store),
		rag.WithEmbeddings(embeddings),
		rag.WithChunkSize(1000),
		rag.WithChunkOverlap(200),
		rag.WithTopK(3),
	)
	if err != nil {
		t.Fatalf("Failed to create RAG: %v", err)
	}

	// Create a query engine
	engine := rag.NewQueryEngine(ragSystem, rag.QueryOptions{
		NumDocuments:    2,
		IncludeMetadata: true,
		PromptTemplate:  "Answer based on: {{context}}\nQuestion: {{query}}",
	})

	// Create documents and convert to chunks
	ctx := context.Background()
	chunks := []*rag.Chunk{
		{
			ID:         "chunk1",
			DocumentID: "doc1",
			Content:    "The capital of France is Paris.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
		{
			ID:         "chunk2",
			DocumentID: "doc2",
			Content:    "The capital of Germany is Berlin.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
	}

	// Generate embeddings for each chunk
	for i := range chunks {
		embedding, err := embeddings.EmbedDocument(ctx, chunks[i].Content)
		if err != nil {
			t.Fatalf("Failed to embed chunk: %v", err)
		}
		chunks[i].Embedding = embedding
	}

	// Add chunks to the store
	err = store.AddChunks(ctx, chunks)
	if err != nil {
		t.Fatalf("Failed to add chunks: %v", err)
	}

	// Create a mock provider for the test
	provider := &MockProvider{
		name: "mock_provider",
	}

	// Add the provider to the context
	ctxWithProvider := context.WithValue(ctx, "llm_provider", provider)

	// Query the engine
	response, err := engine.Query(ctxWithProvider, "What is the capital of France?")
	if err != nil {
		t.Fatalf("Failed to query engine: %v", err)
	}

	// Check the response
	if response == nil {
		t.Fatal("Response is nil")
	}
}

// TestRAGSystem tests the complete RAG system
func TestRAGSystem(t *testing.T) {
	// Create a mock embedding provider
	embeddings := &MockEmbeddingProvider{}

	// Create a mock LLM provider
	provider := &MockProvider{
		name: "mock_provider",
	}

	// Create a RAG system
	system := rag.NewRAGSystem(provider, embeddings)

	// Since the RAGSystem's FileLoader has basePath set to "." by default,
	// we'll create a test file in the current directory

	// Create a unique filename in the current directory
	tempFile := "test_rag_system_" + t.Name() + ".txt"

	// Write content to the file
	content := "The capital of France is Paris. The capital of Germany is Berlin."
	err := ioutil.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Make sure to clean up the file after the test
	defer os.Remove(tempFile)

	// Add the file to the system
	ctx := context.Background()
	err = system.AddFile(ctx, tempFile)
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	// Create a context with the mock provider
	ctxWithProvider := context.WithValue(ctx, "llm_provider", provider)

	// Query the system
	response, err := system.Query(ctxWithProvider, "What is the capital of France?")
	if err != nil {
		t.Fatalf("Failed to query system: %v", err)
	}

	// Check the response
	if response == nil {
		t.Fatal("Response is nil")
	}

	// Set query options
	system.SetQueryOptions(rag.QueryOptions{
		NumDocuments:    1,
		IncludeMetadata: false,
	})

	// Query again with the provider in context
	response, err = system.Query(ctxWithProvider, "What is the capital of Germany?")
	if err != nil {
		t.Fatalf("Failed to query system with options: %v", err)
	}

	// Check the response
	if response == nil {
		t.Fatal("Response is nil")
	}
}

// MockEmbeddingProvider is a mock implementation of the EmbeddingsProvider interface
type MockEmbeddingProvider struct{}

// EmbedQuery generates an embedding for a query
func (p *MockEmbeddingProvider) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	// Generate a simple embedding based on the text length
	embedding := make([]float32, 10)
	for i := range embedding {
		embedding[i] = float32(len(text) % (i + 1))
	}
	return embedding, nil
}

// EmbedDocument generates an embedding for a document
func (p *MockEmbeddingProvider) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	// Generate a simple embedding based on the text length
	embedding := make([]float32, 10)
	for i := range embedding {
		embedding[i] = float32(len(text) % (i + 1))
	}
	return embedding, nil
}

// MockProvider is a mock implementation of the LLMProvider interface
type MockProvider struct {
	name string
}

// Name returns the name of the provider
func (p *MockProvider) Name() string {
	return p.name
}

// Generate generates a response for the given prompt
func (p *MockProvider) Generate(ctx context.Context, prompt *core.Prompt) (*core.Response, error) {
	return &core.Response{
		Text: "Mock response for: " + prompt.Text,
		TokensUsed: &core.TokenUsage{
			Prompt:     10,
			Completion: 10,
			Total:      20,
		},
		FinishReason: "stop",
		ModelInfo: &core.ModelInfo{
			Name:     "mock-model",
			Provider: p.name,
		},
		ProviderInfo: &core.ProviderInfo{
			Name:    p.name,
			Version: "1.0.0",
		},
	}, nil
}

// GenerateStream generates a streaming response for the given prompt
func (p *MockProvider) GenerateStream(ctx context.Context, prompt *core.Prompt) (core.ResponseStream, error) {
	return &MockResponseStream{
		chunks: []string{
			"Mock ",
			"response ",
			"for: ",
			prompt.Text,
		},
	}, nil
}

// MockResponseStream is a mock implementation of the ResponseStream interface
type MockResponseStream struct {
	chunks []string
	index  int
}

// Next returns the next chunk of the response
func (s *MockResponseStream) Next() (*core.ResponseChunk, error) {
	if s.index >= len(s.chunks) {
		return nil, io.EOF
	}

	chunk := &core.ResponseChunk{
		Text:    s.chunks[s.index],
		IsFinal: s.index == len(s.chunks)-1,
	}

	if chunk.IsFinal {
		chunk.FinishReason = "stop"
	}

	s.index++
	return chunk, nil
}

// Close closes the stream
func (s *MockResponseStream) Close() error {
	return nil
}
