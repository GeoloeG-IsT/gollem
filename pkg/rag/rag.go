package rag

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// Document represents a document in the RAG system
type Document struct {
	// ID is a unique identifier for the document
	ID string

	// Content is the text content of the document
	Content string

	// Metadata is additional information about the document
	Metadata map[string]interface{}

	// Embedding is the vector representation of the document
	Embedding []float32
}

// Chunk represents a chunk of a document
type Chunk struct {
	// ID is a unique identifier for the chunk
	ID string

	// DocumentID is the ID of the document this chunk belongs to
	DocumentID string

	// Content is the text content of the chunk
	Content string

	// Metadata is additional information about the chunk
	Metadata map[string]interface{}

	// Embedding is the vector representation of the chunk
	Embedding []float32
}

// RAG represents a Retrieval-Augmented Generation system
type RAG struct {
	// VectorStore is the vector store used for retrieval
	VectorStore VectorStore

	// Embeddings is the embeddings provider
	Embeddings EmbeddingsProvider

	// ChunkSize is the size of each chunk
	ChunkSize int

	// ChunkOverlap is the overlap between chunks
	ChunkOverlap int

	// TopK is the number of chunks to retrieve
	TopK int
}

// RAGOption is a function that configures a RAG
type RAGOption func(*RAG)

// WithVectorStore sets the vector store
func WithVectorStore(vectorStore VectorStore) RAGOption {
	return func(r *RAG) {
		r.VectorStore = vectorStore
	}
}

// WithEmbeddings sets the embeddings provider
func WithEmbeddings(embeddings EmbeddingsProvider) RAGOption {
	return func(r *RAG) {
		r.Embeddings = embeddings
	}
}

// WithChunkSize sets the chunk size
func WithChunkSize(chunkSize int) RAGOption {
	return func(r *RAG) {
		r.ChunkSize = chunkSize
	}
}

// WithChunkOverlap sets the chunk overlap
func WithChunkOverlap(chunkOverlap int) RAGOption {
	return func(r *RAG) {
		r.ChunkOverlap = chunkOverlap
	}
}

// WithTopK sets the number of chunks to retrieve
func WithTopK(topK int) RAGOption {
	return func(r *RAG) {
		r.TopK = topK
	}
}

// NewRAG creates a new RAG system
func NewRAG(options ...RAGOption) (*RAG, error) {
	rag := &RAG{
		ChunkSize:    1000,
		ChunkOverlap: 200,
		TopK:         3,
	}

	for _, option := range options {
		option(rag)
	}

	if rag.VectorStore == nil {
		return nil, errors.New("vector store is required")
	}

	if rag.Embeddings == nil {
		return nil, errors.New("embeddings provider is required")
	}

	return rag, nil
}

// AddDocument adds a document to the RAG system
func (r *RAG) AddDocument(ctx context.Context, document *Document) error {
	// Generate embeddings for the document if not already present
	if document.Embedding == nil {
		embedding, err := r.Embeddings.EmbedDocument(ctx, document.Content)
		if err != nil {
			return fmt.Errorf("failed to embed document: %w", err)
		}
		document.Embedding = embedding
	}

	// Chunk the document
	chunks := r.chunkDocument(document)

	// Generate embeddings for each chunk
	for i := range chunks {
		embedding, err := r.Embeddings.EmbedDocument(ctx, chunks[i].Content)
		if err != nil {
			return fmt.Errorf("failed to embed chunk: %w", err)
		}
		chunks[i].Embedding = embedding
	}

	// Add the chunks to the vector store
	if err := r.VectorStore.AddChunks(ctx, chunks); err != nil {
		return fmt.Errorf("failed to add chunks to vector store: %w", err)
	}

	return nil
}

// AddDocuments adds multiple documents to the RAG system
func (r *RAG) AddDocuments(ctx context.Context, documents []*Document) error {
	for _, document := range documents {
		if err := r.AddDocument(ctx, document); err != nil {
			return err
		}
	}
	return nil
}

// Query retrieves relevant chunks for a query and generates a response
func (r *RAG) Query(ctx context.Context, query string, llm core.LLMProvider) (*core.Response, error) {
	// Retrieve relevant chunks
	chunks, err := r.RetrieveChunks(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve chunks: %w", err)
	}

	// Create a prompt with the retrieved chunks
	prompt := r.createPrompt(query, chunks)

	// Generate a response
	response, err := llm.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	return response, nil
}

// RetrieveChunks retrieves relevant chunks for a query
func (r *RAG) RetrieveChunks(ctx context.Context, query string) ([]*Chunk, error) {
	// Generate embedding for the query
	embedding, err := r.Embeddings.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Retrieve chunks from the vector store
	chunks, err := r.VectorStore.SimilaritySearch(ctx, embedding, r.TopK)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store: %w", err)
	}

	return chunks, nil
}

// chunkDocument chunks a document into smaller pieces
func (r *RAG) chunkDocument(document *Document) []*Chunk {
	content := document.Content
	chunkSize := r.ChunkSize
	chunkOverlap := r.ChunkOverlap

	// If the content is smaller than the chunk size, return a single chunk
	if len(content) <= chunkSize {
		return []*Chunk{
			{
				ID:         fmt.Sprintf("%s-0", document.ID),
				DocumentID: document.ID,
				Content:    content,
				Metadata:   document.Metadata,
			},
		}
	}

	// Split the content into chunks
	var chunks []*Chunk
	for i := 0; i < len(content); i += chunkSize - chunkOverlap {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}

		chunks = append(chunks, &Chunk{
			ID:         fmt.Sprintf("%s-%d", document.ID, i),
			DocumentID: document.ID,
			Content:    content[i:end],
			Metadata:   document.Metadata,
		})

		if end == len(content) {
			break
		}
	}

	return chunks
}

// createPrompt creates a prompt with the retrieved chunks
func (r *RAG) createPrompt(query string, chunks []*Chunk) *core.Prompt {
	var sb strings.Builder

	sb.WriteString("Answer the following question based on the provided context:\n\n")
	sb.WriteString("Context:\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("--- Document %d ---\n", i+1))
		sb.WriteString(chunk.Content)
		sb.WriteString("\n\n")
	}

	sb.WriteString("Question: ")
	sb.WriteString(query)

	return core.NewPrompt(sb.String())
}

// LoadDocumentsFromDirectory loads documents from a directory
func LoadDocumentsFromDirectory(directory string, fileExtensions []string) ([]*Document, error) {
	var documents []*Document

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if the file has one of the specified extensions
		ext := strings.ToLower(filepath.Ext(path))
		if len(fileExtensions) > 0 {
			found := false
			for _, validExt := range fileExtensions {
				if ext == validExt || ext == "."+validExt {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		// Create a document
		relPath, err := filepath.Rel(directory, path)
		if err != nil {
			relPath = path
		}

		document := &Document{
			ID:      relPath,
			Content: string(content),
			Metadata: map[string]interface{}{
				"path":      path,
				"extension": ext,
				"size":      info.Size(),
				"modified":  info.ModTime(),
			},
		}

		documents = append(documents, document)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return documents, nil
}

// LoadDocumentFromFile loads a document from a file
func LoadDocumentFromFile(path string) (*Document, error) {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create a document
	document := &Document{
		ID:      filepath.Base(path),
		Content: string(content),
		Metadata: map[string]interface{}{
			"path":      path,
			"extension": strings.ToLower(filepath.Ext(path)),
		},
	}

	return document, nil
}

// VectorStore is an interface for vector stores
type VectorStore interface {
	// AddChunks adds chunks to the vector store
	AddChunks(ctx context.Context, chunks []*Chunk) error

	// SimilaritySearch searches for chunks similar to the query embedding
	SimilaritySearch(ctx context.Context, embedding []float32, limit int) ([]*Chunk, error)

	// Delete deletes chunks from the vector store
	Delete(ctx context.Context, ids []string) error

	// Clear clears the vector store
	Clear(ctx context.Context) error
}

// EmbeddingsProvider is an interface for embeddings providers
type EmbeddingsProvider interface {
	// EmbedDocument generates an embedding for a document
	EmbedDocument(ctx context.Context, text string) ([]float32, error)

	// EmbedQuery generates an embedding for a query
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

// MemoryVectorStore is an in-memory implementation of VectorStore
type MemoryVectorStore struct {
	chunks     []*Chunk
	embeddings EmbeddingsProvider
	mu         sync.Mutex
}

// NewMemoryVectorStore creates a new in-memory vector store
func NewMemoryVectorStore(embeddings EmbeddingsProvider) *MemoryVectorStore {
	return &MemoryVectorStore{
		chunks:     make([]*Chunk, 0),
		embeddings: embeddings,
	}
}

// AddChunks adds chunks to the vector store
func (s *MemoryVectorStore) AddChunks(ctx context.Context, chunks []*Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add the chunks
	s.chunks = append(s.chunks, chunks...)

	return nil
}

// SimilaritySearch searches for chunks similar to the query embedding
func (s *MemoryVectorStore) SimilaritySearch(ctx context.Context, embedding []float32, limit int) ([]*Chunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate similarity scores
	type chunkScore struct {
		chunk *Chunk
		score float32
	}

	scores := make([]chunkScore, 0, len(s.chunks))
	for _, chunk := range s.chunks {
		score := cosineSimilarity(embedding, chunk.Embedding)
		scores = append(scores, chunkScore{chunk, score})
	}

	// Sort by score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	// Return the top chunks
	result := make([]*Chunk, 0, limit)
	for i := 0; i < limit && i < len(scores); i++ {
		result = append(result, scores[i].chunk)
	}

	return result, nil
}

// Delete deletes chunks from the vector store
func (s *MemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a map of IDs to delete
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	// Filter out the chunks to delete
	filtered := make([]*Chunk, 0, len(s.chunks))
	for _, chunk := range s.chunks {
		if !idMap[chunk.ID] {
			filtered = append(filtered, chunk)
		}
	}

	s.chunks = filtered

	return nil
}

// Clear clears the vector store
func (s *MemoryVectorStore) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.chunks = make([]*Chunk, 0)

	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// RemoteVectorStore is a vector store that uses a remote API
type RemoteVectorStore struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

// NewRemoteVectorStore creates a new remote vector store
func NewRemoteVectorStore(endpoint, apiKey string) *RemoteVectorStore {
	return &RemoteVectorStore{
		endpoint: endpoint,
		apiKey:   apiKey,
		client:   &http.Client{},
	}
}

// AddChunks adds chunks to the vector store
func (s *RemoteVectorStore) AddChunks(ctx context.Context, chunks []*Chunk) error {
	// In a real implementation, this would send the chunks to the remote API
	return nil
}

// SimilaritySearch searches for chunks similar to the query embedding
func (s *RemoteVectorStore) SimilaritySearch(ctx context.Context, embedding []float32, limit int) ([]*Chunk, error) {
	// In a real implementation, this would query the remote API
	return nil, nil
}

// Delete deletes chunks from the vector store
func (s *RemoteVectorStore) Delete(ctx context.Context, ids []string) error {
	// In a real implementation, this would delete the chunks from the remote API
	return nil
}

// Clear clears the vector store
func (s *RemoteVectorStore) Clear(ctx context.Context) error {
	// In a real implementation, this would clear the remote vector store
	return nil
}
