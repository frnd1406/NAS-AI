package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nas-ai/api/src/database"
	"github.com/sirupsen/logrus"
)

type embedQueryResponse struct {
	Embedding []float64 `json:"embedding"`
}

type searchResult struct {
	FilePath   string  `json:"file_path"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
}

// SearchHandler handles advanced hybrid search combining:
// - Semantic similarity (AI embeddings)
// - Full-text keyword matching
// - Per-word match counting with frequency bonus
// - Exact phrase matching
// - Filename matching
// - Normalized scoring (0-100%)
func SearchHandler(db *database.DB, aiServiceURL string, httpClient *http.Client, logger *logrus.Logger) gin.HandlerFunc {
	client := httpClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	baseURL := strings.TrimRight(aiServiceURL, "/")

	return func(c *gin.Context) {
		query := strings.TrimSpace(c.Query("q"))
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing query parameter 'q'"})
			return
		}

		embedding, err := fetchEmbedding(c.Request.Context(), client, baseURL+"/embed_query", query)
		if err != nil {
			logger.WithError(err).Warn("Failed to get embedding, falling back to text-only search")
			// Proceed with empty embedding (will skip vector similarity part in SQL)
			embedding = make([]float64, 1024)
		}

		// Convert embedding to pgvector format
		parts := make([]string, len(embedding))
		for i, v := range embedding {
			parts[i] = strconv.FormatFloat(v, 'f', -1, 64)
		}
		embeddingStr := "[" + strings.Join(parts, ",") + "]"

		// Prepare search terms
		words := strings.Fields(strings.ToLower(query))
		wordCount := len(words)

		// Build queries
		tsQueryOr := strings.Join(words, " | ") // Any word matches

		// Build LIKE patterns for each word (for counting individual matches)
		// We'll pass all words as a single array-like parameter
		wordsForLike := strings.Join(words, "|")

		// Advanced Hybrid Search Query - NATURAL LANGUAGE OPTIMIZED
		// Scoring breakdown:
		// - Semantic score:  0.0 - 0.80 (80% max) - AI understands meaning
		// - All words found: 0.0 - 0.10 (10% bonus)
		// - Per-word matches: 0.0 - 0.05 (5% based on % of words found)
		// - Exact phrase:    0.0 - 0.05 (5% bonus)
		// - Filename match:  0.0 - 0.05 (5% bonus)
		// Total possible: ~100%
		rows, err := db.QueryContext(c.Request.Context(), `
			WITH search_words AS (
				SELECT unnest(string_to_array($4::text, '|')) as word
			),
			word_matches AS (
				SELECT 
					fe.metadata->>'file_path' as file_path,
					fe.content,
					fe.embedding,
					COUNT(DISTINCT sw.word) as matched_word_count,
					-- Count total occurrences of all search words
					SUM(
						(length(lower(fe.content)) - length(replace(lower(fe.content), sw.word, ''))) 
						/ GREATEST(length(sw.word), 1)
					) as total_word_occurrences
				FROM file_embeddings fe
				CROSS JOIN search_words sw
				WHERE (fe.metadata->>'file_path') LIKE '/mnt/data/%'
				  AND (fe.metadata->>'file_path') NOT LIKE '%/.trash/%'
				  AND lower(fe.content) LIKE '%' || sw.word || '%'
				GROUP BY fe.metadata->>'file_path', fe.content, fe.embedding
			),
			all_docs AS (
				SELECT 
					fe.metadata->>'file_path' as file_path,
					content,
					embedding,
					0 as matched_word_count,
					0 as total_word_occurrences
				FROM file_embeddings fe
				WHERE (fe.metadata->>'file_path') LIKE '/mnt/data/%'
				  AND (fe.metadata->>'file_path') NOT LIKE '%/.trash/%'
				  AND (fe.metadata->>'file_path') NOT IN (SELECT file_path FROM word_matches)
			),
			combined AS (
				SELECT * FROM word_matches
				UNION ALL
				SELECT * FROM all_docs
			),
			scored AS (
				SELECT 
					file_path,
					content,
					-- Semantic score: HIGH weight for natural language understanding
					(1 - (embedding <=> $1::vector)) * 0.85 as semantic_score,
					
					-- All words bonus: 0.25 if ALL search words are found
					CASE 
						WHEN matched_word_count >= $5::int THEN 0.08
						ELSE 0
					END as all_words_bonus,
					
					-- Per-word match score: proportional to words found (max 0.05)
					(matched_word_count::float / GREATEST($5::int, 1)) * 0.05 as word_match_score,
					
					-- Word frequency bonus: more occurrences = higher score (max 0.02)
					LEAST(0.02, (total_word_occurrences::float / 10) * 0.02) as frequency_bonus,
					
					-- Exact phrase bonus: 0.10 if exact phrase appears
					CASE 
						WHEN lower(content) LIKE '%' || lower($3::text) || '%' THEN 0.05
						ELSE 0
					END as exact_phrase_bonus,
					
					-- Filename bonus: 0.03 if filename contains search terms
					CASE 
						WHEN lower(file_path) LIKE '%' || lower(replace($3::text, ' ', '%')) || '%' THEN 0.03
						ELSE 0
					END as filename_bonus,
					
					-- Full-text search rank bonus (for relevance ordering)
					CASE 
						WHEN to_tsvector('simple', content) @@ to_tsquery('simple', $2::text)
						THEN ts_rank(to_tsvector('simple', content), to_tsquery('simple', $2::text)) * 0.02
						ELSE 0
					END as fts_bonus
				FROM combined
			)
			SELECT 
				file_path,
				content,
				-- Final score: sum of all components, capped at 1.0
				LEAST(1.0, 
					semantic_score + 
					all_words_bonus + 
					word_match_score + 
					frequency_bonus +
					exact_phrase_bonus + 
					filename_bonus +
					fts_bonus
				) as similarity
			FROM scored
			ORDER BY similarity DESC
			LIMIT 10;
		`, embeddingStr, tsQueryOr, query, wordsForLike, wordCount)
		if err != nil {
			logger.WithError(err).Error("Failed to run advanced hybrid search query")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "database query failed"})
			return
		}
		defer rows.Close()

		results := make([]searchResult, 0, 10)
		for rows.Next() {
			var r searchResult
			if scanErr := rows.Scan(&r.FilePath, &r.Content, &r.Similarity); scanErr != nil {
				logger.WithError(scanErr).Error("Failed to scan search result row")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read results"})
				return
			}
			results = append(results, r)
		}
		if err := rows.Err(); err != nil {
			logger.WithError(err).Error("Row iteration error during search")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read results"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"query":   query,
			"results": results,
		})
	}
}

func fetchEmbedding(ctx context.Context, client *http.Client, url, query string) ([]float64, error) {
	payload := map[string]string{"text": query}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		return nil, fmt.Errorf("encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ai agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("ai agent status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed embedQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if len(parsed.Embedding) == 0 {
		return nil, fmt.Errorf("empty embedding")
	}

	return parsed.Embedding, nil
}
