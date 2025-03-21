package rag_test

import (
	"context"
	"testing"

	"github.com/user/gollem/pkg/core"
	"github.com/user/gollem/pkg/rag"
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
	splitter := rag.NewCharacterTextSplitter(
		rag.WithChunkSize(10),
		rag.WithChunkOverlap(2),
	)
	
	// Split text
	text := "This is a test document that needs to be split into chunks."
	chunks := splitter.SplitText(text)
	
	// Check the chunks
	if len(chunks) == 0 {
		t.Fatal("No chunks created")
	}
	
	// Check that each chunk is not longer than the chunk size
	for i, chunk := range chunks {
		if len(chunk) > 10 && i < len(chunks)-1 {
			t.Fatalf("Chunk %d is too long: %s", i, chunk)
		}
	}
	
	// Test document splitting
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
	
	// Create documents
	docs := []*rag.Document{
		{
			ID:      "doc1",
			Content: "This is the first document.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
		{
			ID:      "doc2",
			Content: "This is the second document.",
			Metadata: map[string]interface{}{
				"source": "test",
			},
		},
	}
	
	// Add documents to the store
	ctx := context.Background()
	err := store.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}
	
	// Search for similar documents
	results, err := store.SimilaritySearch(ctx, "first document", 1)
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
	
	// Try to search again
	_, err = store.SimilaritySearch(ctx, "first document", 1)
	if err == nil {
		t.Fatal("No error when searching empty store")
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
	
	// Create a text splitter
	splitter := rag.NewCharacterTextSplitter()
	
	// Create a RAG
	ragSystem := rag.NewRAG(store, provider, splitter)
	
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
	err := ragSystem.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}
	
	// Query the RAG
	response, err := ragSystem.Query(ctx, "What is the capital of France?", 1)
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
	
	// Create a mock LLM provider
	provider := &MockProvider{
		name: "mock_provider",
	}
	
	// Create a text splitter
	splitter := rag.NewCharacterTextSplitter()
	
	// Create a RAG
	ragSystem := rag.NewRAG(store, provider, splitter)
	
	// Create a query engine
	engine := rag.NewQueryEngine(ragSystem, rag.QueryOptions{
		NumDocuments:   2,
		IncludeMetadata: true,
		PromptTemplate: "Answer based on: {{context}}\nQuestion: {{query}}",
	})
	
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
	err := store.AddDocuments(ctx, docs)
	if err != nil {
		t.Fatalf("Failed to add documents: %v", err)
	}
	
	// Query the engine
	response, err := engine.Query(ctx, "What is the capital of France?")
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
	
	// Create a temporary file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")
	
	// Write content to the file
	content := "The capital of France is Paris. The capital of Germany is Berlin."
	err := ioutil.WriteFile(tempFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}
	
	// Add the file to the system
	ctx := context.Background()
	err = system.AddFile(ctx, tempFile)
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	
	// Query the system
	response, err := system.Query(ctx, "What is the capital of France?")
	if err != nil {
		t.Fatalf("Failed to query system: %v", err)
	}
	
	// Check the response
	if response == nil {
		t.Fatal("Response is nil")
	}
	
	// Set query options
	system.SetQueryOptions(rag.QueryOptions{
		NumDocuments:   1,
		IncludeMetadata: false,
	})
	
	// Query again
	response, err = system.Query(ctx, "What is the capital of Germany?")
	if err != nil {
		t.Fatalf("Failed to query system with options: %v", err)
	}
	
	// Check the response
	if response == nil {
		t.Fatal("Response is nil")
	}
}

// MockEmbeddingProvider is a mock implementation of the EmbeddingProvider interface
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

// EmbedDocuments generates embeddings for documents
func (p *MockEmbeddingProvider) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding := make([]float32, 10)
		for j := range embedding {
			embedding[j] = float32(len(text) % (j + 1))
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
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
