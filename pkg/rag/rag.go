package rag

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// Document represents a document in the RAG system
type Document struct {
	// ID is a unique identifier for the document
	ID string

	// Content is the text content of the document
	Content string

	// Metadata contains additional information about the document
	Metadata map[string]interface{}
}

// DocumentLoader loads documents from various sources
type DocumentLoader interface {
	// LoadDocument loads a document from a source
	LoadDocument(ctx context.Context, source string) (*Document, error)

	// LoadDocuments loads multiple documents from a source
	LoadDocuments(ctx context.Context, source string) ([]*Document, error)
}

// TextSplitter splits text into chunks
type TextSplitter interface {
	// SplitText splits text into chunks
	SplitText(text string, options ...SplitOption) []string

	// SplitDocument splits a document into multiple documents
	SplitDocument(doc *Document, options ...SplitOption) []*Document
}

// SplitOption configures a text splitter
type SplitOption func(interface{})

// WithChunkSize sets the chunk size
func WithChunkSize(size int) SplitOption {
	return func(s interface{}) {
		if ts, ok := s.(*CharacterTextSplitter); ok {
			ts.chunkSize = size
		}
	}
}

// WithChunkOverlap sets the chunk overlap
func WithChunkOverlap(overlap int) SplitOption {
	return func(s interface{}) {
		if ts, ok := s.(*CharacterTextSplitter); ok {
			ts.chunkOverlap = overlap
		}
	}
}

// CharacterTextSplitter splits text by character count
type CharacterTextSplitter struct {
	chunkSize    int
	chunkOverlap int
	separator    string
}

// NewCharacterTextSplitter creates a new character text splitter
func NewCharacterTextSplitter(options ...SplitOption) *CharacterTextSplitter {
	splitter := &CharacterTextSplitter{
		chunkSize:    1000,
		chunkOverlap: 200,
		separator:    "\n",
	}

	for _, option := range options {
		option(splitter)
	}

	return splitter
}

// SplitText splits text into chunks
func (s *CharacterTextSplitter) SplitText(text string, options ...SplitOption) []string {
	// Apply options
	for _, option := range options {
		option(s)
	}

	// Split the text by separator
	parts := strings.Split(text, s.separator)

	// Combine parts into chunks
	var chunks []string
	var currentChunk strings.Builder
	var currentSize int

	for _, part := range parts {
		// If adding this part would exceed the chunk size, start a new chunk
		if currentSize+len(part)+len(s.separator) > s.chunkSize && currentSize > 0 {
			chunks = append(chunks, currentChunk.String())

			// Start a new chunk with overlap
			if s.chunkOverlap > 0 {
				// Get the last part of the previous chunk for overlap
				prevChunk := currentChunk.String()
				overlapStart := len(prevChunk) - s.chunkOverlap
				if overlapStart < 0 {
					overlapStart = 0
				}
				overlap := prevChunk[overlapStart:]

				currentChunk = strings.Builder{}
				currentChunk.WriteString(overlap)
				currentSize = len(overlap)
			} else {
				currentChunk = strings.Builder{}
				currentSize = 0
			}
		}

		// Add the part to the current chunk
		if currentSize > 0 {
			currentChunk.WriteString(s.separator)
			currentSize += len(s.separator)
		}
		currentChunk.WriteString(part)
		currentSize += len(part)
	}

	// Add the last chunk if it's not empty
	if currentSize > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// SplitDocument splits a document into multiple documents
func (s *CharacterTextSplitter) SplitDocument(doc *Document, options ...SplitOption) []*Document {
	// Split the text
	chunks := s.SplitText(doc.Content, options...)

	// Create a new document for each chunk
	docs := make([]*Document, len(chunks))
	for i, chunk := range chunks {
		docs[i] = &Document{
			ID:       fmt.Sprintf("%s_chunk_%d", doc.ID, i),
			Content:  chunk,
			Metadata: doc.Metadata,
		}
	}

	return docs
}

// FileLoader loads documents from files
type FileLoader struct {
	// BasePath is the base path for relative file paths
	BasePath string
}

// NewFileLoader creates a new file loader
func NewFileLoader(basePath string) *FileLoader {
	return &FileLoader{
		BasePath: basePath,
	}
}

// LoadDocument loads a document from a file
func (l *FileLoader) LoadDocument(ctx context.Context, source string) (*Document, error) {
	// Check if the context is canceled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Resolve the path
	path := source
	if !filepath.IsAbs(path) {
		path = filepath.Join(l.BasePath, path)
	}

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read the file
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create a document
	doc := &Document{
		ID:      filepath.Base(path),
		Content: string(content),
		Metadata: map[string]interface{}{
			"path": path,
			"type": "file",
		},
	}

	return doc, nil
}

// LoadDocuments loads multiple documents from a directory
func (l *FileLoader) LoadDocuments(ctx context.Context, source string) ([]*Document, error) {
	// Check if the context is canceled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Resolve the path
	path := source
	if !filepath.IsAbs(path) {
		path = filepath.Join(l.BasePath, path)
	}

	// Get file info
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	// If it's a file, load it as a single document
	if !info.IsDir() {
		doc, err := l.LoadDocument(ctx, path)
		if err != nil {
			return nil, err
		}
		return []*Document{doc}, nil
	}

	// If it's a directory, load all files in it
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var docs []*Document
	for _, file := range files {
		// Skip directories
		if file.IsDir() {
			continue
		}

		// Load the document
		doc, err := l.LoadDocument(ctx, filepath.Join(path, file.Name()))
		if err != nil {
			return nil, err
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// VectorStore is an interface for storing and retrieving document vectors
type VectorStore interface {
	// AddDocuments adds documents to the store
	AddDocuments(ctx context.Context, docs []*Document) error

	// SimilaritySearch searches for similar documents
	SimilaritySearch(ctx context.Context, query string, k int) ([]*Document, error)

	// Clear removes all documents from the store
	Clear(ctx context.Context) error
}

// EmbeddingProvider generates embeddings for text
type EmbeddingProvider interface {
	// EmbedQuery generates an embedding for a query
	EmbedQuery(ctx context.Context, text string) ([]float32, error)

	// EmbedDocuments generates embeddings for documents
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}

// MemoryVectorStore is an in-memory implementation of VectorStore
type MemoryVectorStore struct {
	embeddings EmbeddingProvider
	documents  []*Document
	vectors    [][]float32
}

// NewMemoryVectorStore creates a new memory vector store
func NewMemoryVectorStore(embeddings EmbeddingProvider) *MemoryVectorStore {
	return &MemoryVectorStore{
		embeddings: embeddings,
		documents:  make([]*Document, 0),
		vectors:    make([][]float32, 0),
	}
}

// AddDocuments adds documents to the store
func (s *MemoryVectorStore) AddDocuments(ctx context.Context, docs []*Document) error {
	// Extract the text from the documents
	texts := make([]string, len(docs))
	for i, doc := range docs {
		texts[i] = doc.Content
	}

	// Generate embeddings for the documents
	vectors, err := s.embeddings.EmbedDocuments(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to embed documents: %w", err)
	}

	// Add the documents and vectors to the store
	s.documents = append(s.documents, docs...)
	s.vectors = append(s.vectors, vectors...)

	return nil
}

// SimilaritySearch searches for similar documents
func (s *MemoryVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]*Document, error) {
	// Check if we have any documents
	if len(s.documents) == 0 {
		return nil, errors.New("no documents in the store")
	}

	// Generate an embedding for the query
	queryVector, err := s.embeddings.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Calculate the similarity between the query and each document
	similarities := make([]float32, len(s.vectors))
	for i, vector := range s.vectors {
		similarities[i] = cosineSimilarity(queryVector, vector)
	}

	// Find the k most similar documents
	indices := topK(similarities, k)

	// Return the documents
	results := make([]*Document, len(indices))
	for i, idx := range indices {
		results[i] = s.documents[idx]
	}

	return results, nil
}

// Clear removes all documents from the store
func (s *MemoryVectorStore) Clear(ctx context.Context) error {
	s.documents = make([]*Document, 0)
	s.vectors = make([][]float32, 0)
	return nil
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	// Calculate the dot product
	var dotProduct float32
	for i := range a {
		dotProduct += a[i] * b[i]
	}

	// Calculate the magnitudes
	var magA, magB float32
	for i := range a {
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	magA = float32(math.Sqrt(float64(magA)))
	magB = float32(math.Sqrt(float64(magB)))

	// Calculate the cosine similarity
	if magA == 0 || magB == 0 {
		return 0
	}
	return dotProduct / (magA * magB)
}

// topK returns the indices of the k largest elements in the slice
func topK(values []float32, k int) []int {
	// Create a slice of indices
	indices := make([]int, len(values))
	for i := range indices {
		indices[i] = i
	}

	// Sort the indices by the values
	sort.Slice(indices, func(i, j int) bool {
		return values[indices[i]] > values[indices[j]]
	})

	// Return the top k indices
	if k > len(indices) {
		k = len(indices)
	}
	return indices[:k]
}

// RAG implements Retrieval Augmented Generation
type RAG struct {
	vectorStore VectorStore
	llmProvider core.LLMProvider
	textSplitter TextSplitter
}

// NewRAG creates a new RAG instance
func NewRAG(vectorStore VectorStore, llmProvider core.LLMProvider, textSplitter TextSplitter) *RAG {
	return &RAG{
		vectorStore: vectorStore,
		llmProvider: llmProvider,
		textSplitter: textSplitter,
	}
}

// AddDocuments adds documents to the RAG system
func (r *RAG) AddDocuments(ctx context.Context, docs []*Document) error {
	// Split the documents
	var splitDocs []*Document
	for _, doc := range docs {
		splitDocs = append(splitDocs, r.textSplitter.SplitDocument(doc)...)
	}

	// Add the documents to the vector store
	return r.vectorStore.AddDocuments(ctx, splitDocs)
}

// Query performs a RAG query
func (r *RAG) Query(ctx context.Context, query string, k int) (*core.Response, error) {
	// Search for similar documents
	docs, err := r.vectorStore.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, fmt.Errorf("failed to search for similar documents: %w", err)
	}

	// Create a prompt with the retrieved documents
	var promptBuilder strings.Builder
	promptBuilder.WriteString("Answer the question based on the following context:\n\n")
	
	for i, doc := range docs {
		promptBuilder.WriteString(fmt.Sprintf("Context %d:\n%s\n\n", i+1, doc.Content))
	}
	
	promptBuilder.WriteString("Question: " + query)
	
	prompt := core.NewPrompt(promptBuilder.String())
	
	// Generate a response
	return r.llmProvider.Generate(ctx, prompt)
}
