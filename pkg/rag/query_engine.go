package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/GeoloeG-IsT/gollem/pkg/core"
)

// Embeddings implements the EmbeddingProvider interface for various embedding models
type Embeddings struct {
	provider core.LLMProvider
	model    string
	dimension int
}

// NewEmbeddings creates a new embeddings provider
func NewEmbeddings(provider core.LLMProvider, model string, dimension int) *Embeddings {
	return &Embeddings{
		provider: provider,
		model:    model,
		dimension: dimension,
	}
}

// EmbedQuery generates an embedding for a query
func (e *Embeddings) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	// In a real implementation, this would call the provider's API to generate an embedding
	// For simplicity, we're just returning a random embedding
	return e.generateRandomEmbedding(), nil
}

// EmbedDocuments generates embeddings for documents
func (e *Embeddings) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	// In a real implementation, this would call the provider's API to generate embeddings
	// For simplicity, we're just returning random embeddings
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = e.generateRandomEmbedding()
	}
	return embeddings, nil
}

// generateRandomEmbedding generates a random embedding for testing
func (e *Embeddings) generateRandomEmbedding() []float32 {
	// Create a deterministic embedding based on the text
	embedding := make([]float32, e.dimension)
	for i := range embedding {
		embedding[i] = float32(math.Sin(float64(i) * 0.1))
	}
	return embedding
}

// QueryEngine performs RAG queries with different strategies
type QueryEngine struct {
	rag *RAG
	options QueryOptions
}

// QueryOptions configures the query engine
type QueryOptions struct {
	// NumDocuments is the number of documents to retrieve
	NumDocuments int
	
	// IncludeMetadata indicates whether to include document metadata in the prompt
	IncludeMetadata bool
	
	// PromptTemplate is the template for the prompt
	PromptTemplate string
}

// NewQueryEngine creates a new query engine
func NewQueryEngine(rag *RAG, options QueryOptions) *QueryEngine {
	// Set default options
	if options.NumDocuments == 0 {
		options.NumDocuments = 3
	}
	
	if options.PromptTemplate == "" {
		options.PromptTemplate = "Answer the question based on the following context:\n\n{{context}}\n\nQuestion: {{query}}"
	}
	
	return &QueryEngine{
		rag: rag,
		options: options,
	}
}

// Query performs a RAG query
func (e *QueryEngine) Query(ctx context.Context, query string) (*core.Response, error) {
	// Search for similar documents
	docs, err := e.rag.vectorStore.SimilaritySearch(ctx, query, e.options.NumDocuments)
	if err != nil {
		return nil, fmt.Errorf("failed to search for similar documents: %w", err)
	}
	
	// Create a prompt with the retrieved documents
	var contextBuilder strings.Builder
	for i, doc := range docs {
		contextBuilder.WriteString(fmt.Sprintf("Context %d:\n%s\n\n", i+1, doc.Content))
		
		// Include metadata if requested
		if e.options.IncludeMetadata && len(doc.Metadata) > 0 {
			contextBuilder.WriteString("Metadata:\n")
			for k, v := range doc.Metadata {
				contextBuilder.WriteString(fmt.Sprintf("%s: %v\n", k, v))
			}
			contextBuilder.WriteString("\n")
		}
	}
	
	// Replace template variables
	promptText := e.options.PromptTemplate
	promptText = strings.ReplaceAll(promptText, "{{context}}", contextBuilder.String())
	promptText = strings.ReplaceAll(promptText, "{{query}}", query)
	
	prompt := core.NewPrompt(promptText)
	
	// Generate a response
	return e.rag.llmProvider.Generate(ctx, prompt)
}

// DocumentStore manages documents in the RAG system
type DocumentStore struct {
	documents map[string]*Document
}

// NewDocumentStore creates a new document store
func NewDocumentStore() *DocumentStore {
	return &DocumentStore{
		documents: make(map[string]*Document),
	}
}

// AddDocument adds a document to the store
func (s *DocumentStore) AddDocument(doc *Document) {
	s.documents[doc.ID] = doc
}

// GetDocument gets a document from the store
func (s *DocumentStore) GetDocument(id string) (*Document, bool) {
	doc, exists := s.documents[id]
	return doc, exists
}

// RemoveDocument removes a document from the store
func (s *DocumentStore) RemoveDocument(id string) {
	delete(s.documents, id)
}

// GetDocuments gets all documents from the store
func (s *DocumentStore) GetDocuments() []*Document {
	docs := make([]*Document, 0, len(s.documents))
	for _, doc := range s.documents {
		docs = append(docs, doc)
	}
	return docs
}

// RAGPipeline combines document loading, splitting, and indexing
type RAGPipeline struct {
	loader      DocumentLoader
	splitter    TextSplitter
	vectorStore VectorStore
}

// NewRAGPipeline creates a new RAG pipeline
func NewRAGPipeline(loader DocumentLoader, splitter TextSplitter, vectorStore VectorStore) *RAGPipeline {
	return &RAGPipeline{
		loader:      loader,
		splitter:    splitter,
		vectorStore: vectorStore,
	}
}

// ProcessFile processes a file and adds it to the vector store
func (p *RAGPipeline) ProcessFile(ctx context.Context, path string) error {
	// Load the document
	doc, err := p.loader.LoadDocument(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to load document: %w", err)
	}
	
	// Split the document
	splitDocs := p.splitter.SplitDocument(doc)
	
	// Add the documents to the vector store
	return p.vectorStore.AddDocuments(ctx, splitDocs)
}

// ProcessDirectory processes a directory and adds all files to the vector store
func (p *RAGPipeline) ProcessDirectory(ctx context.Context, path string) error {
	// Load the documents
	docs, err := p.loader.LoadDocuments(ctx, path)
	if err != nil {
		return fmt.Errorf("failed to load documents: %w", err)
	}
	
	// Split the documents
	var splitDocs []*Document
	for _, doc := range docs {
		splitDocs = append(splitDocs, p.splitter.SplitDocument(doc)...)
	}
	
	// Add the documents to the vector store
	return p.vectorStore.AddDocuments(ctx, splitDocs)
}

// RAGSystem combines all RAG components into a complete system
type RAGSystem struct {
	pipeline    *RAGPipeline
	queryEngine *QueryEngine
	documentStore *DocumentStore
}

// NewRAGSystem creates a new RAG system
func NewRAGSystem(provider core.LLMProvider, embeddingProvider EmbeddingProvider) *RAGSystem {
	// Create the vector store
	vectorStore := NewMemoryVectorStore(embeddingProvider)
	
	// Create the text splitter
	splitter := NewCharacterTextSplitter()
	
	// Create the document loader
	loader := NewFileLoader(".")
	
	// Create the pipeline
	pipeline := NewRAGPipeline(loader, splitter, vectorStore)
	
	// Create the RAG
	rag := NewRAG(vectorStore, provider, splitter)
	
	// Create the query engine
	queryEngine := NewQueryEngine(rag, QueryOptions{})
	
	// Create the document store
	documentStore := NewDocumentStore()
	
	return &RAGSystem{
		pipeline:    pipeline,
		queryEngine: queryEngine,
		documentStore: documentStore,
	}
}

// AddFile adds a file to the RAG system
func (s *RAGSystem) AddFile(ctx context.Context, path string) error {
	return s.pipeline.ProcessFile(ctx, path)
}

// AddDirectory adds a directory to the RAG system
func (s *RAGSystem) AddDirectory(ctx context.Context, path string) error {
	return s.pipeline.ProcessDirectory(ctx, path)
}

// Query performs a RAG query
func (s *RAGSystem) Query(ctx context.Context, query string) (*core.Response, error) {
	return s.queryEngine.Query(ctx, query)
}

// SetQueryOptions sets the query options
func (s *RAGSystem) SetQueryOptions(options QueryOptions) {
	s.queryEngine = NewQueryEngine(s.queryEngine.rag, options)
}
