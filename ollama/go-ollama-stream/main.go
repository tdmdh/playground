package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

type GenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

func Embedding(text string) ([]float64, error) {
	requestData := EmbeddingRequest{
		Model:  "nomic-embed-text",
		Prompt: text,
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(requestData)

	respone, err := http.Post("http://localhost:11434/api/embeddings", "application/json", &buf)
	if err != nil {
		return nil, err
	}
	defer respone.Body.Close()

	var e EmbeddingResponse
	json.NewDecoder(respone.Body).Decode(&e)
	return e.Embedding, err
}

func WordFrequency(sentence string) map[string]float64 {
	words := strings.Fields(strings.ToLower(sentence))
	freq := make(map[string]float64)
	for _, w := range words {
		freq[w]++
	}
	return freq
}

func CosineSimilarity(a, b map[string]float64) float64 {
	var dot, magA, magB float64
	for word, countA := range a {
		if countB, ok := b[word]; ok {
			dot += countA * countB
		}
		magA += countA * countA
	}
	for _, countB := range b {
		magB += countB * countB
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

func CosineSimilarityDense(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	for magA == 0 || magB == 0 {
		return 0.0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

func BuildRAGPrompt(userQuestion string, bestChunk string) string {
	promptTemplate := `You are a helpful assistant. Answer the question based ONLY on the following context.

Context:
%s

Question:
%s

Answer:`
	return fmt.Sprintf(promptTemplate, bestChunk, userQuestion)
}

func streamLLM(prompt string) {
	reqData := GenerateRequest{
		Model:  "llama3",
		Prompt: prompt,
		Stream: true,
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(reqData)

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", &buf)
	if err != nil {
		fmt.Println("Error connecting to LLM:", err)
		return
	}
	defer resp.Body.Close()

	fmt.Print("\n LLM Answer: ")
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var genResp GenerateResponse
		json.Unmarshal(scanner.Bytes(), &genResp)
		fmt.Print(genResp.Response)

		if genResp.Done {
			break
		}
	}
	fmt.Println()
}

func main0() {
	document := []string{
		"The Eiffel Tower is located in Paris, France.",
		"Go is an open source programming language supported by Google.",
		"The Middlemist's Red is widely considered the rarest flower in the world.",
		"The mitochondria is the powerhouse of the cell.",
	}
	userQuery := "What is the rarest flower?"
	fmt.Println("User Query:", userQuery)

	queryVec, err := Embedding(userQuery)
	if err != nil {
		fmt.Println("Failed to embed query:", err)
		return
	}

	var bestMatch string
	var highestScore float64 = -1.0

	fmt.Println("Searching knowladge base..")
	for _, doc := range document {
		docVec, _ := Embedding(doc)
		score := CosineSimilarityDense(queryVec, docVec)

		fmt.Printf(" - Score: %.4f | Doc: %s\n", score, doc)

		if score > highestScore {
			highestScore = score
			bestMatch = doc
		}
	}

	fmt.Printf("\n Best match found (Score: %.4f): %s\n", highestScore, bestMatch)

	promptTemplate := `You are a helpful assistant. Answer the question based ONLY on the following context.

Context:
%s

Question:
%s

Answer:`
	finalPrompt := fmt.Sprintf(promptTemplate, bestMatch, userQuery)

	streamLLM(finalPrompt)
}

func main() {
	ctx := context.Background()

	connStr := "postgres://postgres:postgres@localhost:5432/rag?sslmode=disable"
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		fmt.Printf("Unable to conn", err)
		return
	}
	defer conn.Close(ctx)

	mdv := make([]float32, 768)
	mdv[0] = 0.5
	mqv := make([]float32, 768)
	mqv[0] = 0.45

	fmt.Println("Inserting document...")
	_, err = conn.Exec(ctx,
		"INSERT INTO documents (content, embedding) VALUES ($1, $2)",
		"The powerhouse of the cell is the mitochondria.",
		pgvector.NewVector(mdv),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Searching for closest match...")
	var retrievedContent string

	err = conn.QueryRow(ctx, `
		SELECT content 
		FROM documents 
		ORDER BY embedding <=> $1 
		LIMIT 1
	`, pgvector.NewVector(mqv)).Scan(&retrievedContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Found Match:", retrievedContent)
}
