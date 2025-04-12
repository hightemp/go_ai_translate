package translator

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	APIKey     string
	ToLang     string
	ChunkSize  int
	Model      string
	Verbose    bool
	MaxRetries int
}

type Translator struct {
	config Config
}

func NewTranslator(config Config) *Translator {
	return &Translator{
		config: config,
	}
}

func (t *Translator) TranslateFile(inputPath, outputPath string) error {

	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	chunks := t.splitIntoChunks(string(content))
	if t.config.Verbose {
		fmt.Printf("Split content into %d chunks\n", len(chunks))
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	writer := bufio.NewWriter(outputFile)
	defer writer.Flush()

	for i, chunk := range chunks {
		if t.config.Verbose {
			fmt.Printf("Translating chunk %d of %d (size: %d characters, ~%d tokens)\n",
				i+1, len(chunks), len(chunk), len(chunk)/4)
		}

		var translatedChunk string
		var chunkErr error
		maxRetries := t.config.MaxRetries
		if maxRetries <= 0 {
			maxRetries = 3
		}
		retryDelay := 2 * time.Second

		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				if t.config.Verbose {
					fmt.Printf("Retrying chunk %d translation (attempt %d/%d) after error: %v\n",
						i+1, attempt+1, maxRetries, chunkErr)
				}
				time.Sleep(retryDelay)

				retryDelay *= 2
			}

			translatedChunk, chunkErr = t.translateChunk(chunk)
			if chunkErr == nil {
				break
			}
		}

		if chunkErr != nil {
			return fmt.Errorf("failed to translate chunk %d after %d attempts: %w",
				i+1, maxRetries, chunkErr)
		}

		if _, err := writer.WriteString(translatedChunk); err != nil {
			return fmt.Errorf("failed to write translated chunk to output file: %w", err)
		}

		if i < len(chunks)-1 && !strings.HasSuffix(translatedChunk, "\n") {
			writer.WriteString("\n")
		}

		writer.Flush()

		if i < len(chunks)-1 {
			delay := 10 * time.Millisecond
			if len(chunk) > 1000 {

				additionalDelay := time.Duration(len(chunk)/1000) * 300 * time.Millisecond
				if additionalDelay > 1500*time.Millisecond {
					additionalDelay = 1500 * time.Millisecond
				}
				delay += additionalDelay
			}

			if t.config.Verbose {
				fmt.Printf("Waiting %v before next chunk...\n", delay)
			}
			time.Sleep(delay)
		}
	}

	if t.config.Verbose {
		fmt.Printf("Translation completed successfully\n")
	}

	return nil
}

func (t *Translator) splitIntoChunks(text string) []string {

	if text == "" {
		return []string{}
	}

	estimatedTokens := len(text) / 4

	if estimatedTokens <= t.config.ChunkSize {
		return []string{text}
	}

	effectiveChunkSize := int(float64(t.config.ChunkSize) * 0.8)
	if effectiveChunkSize < 100 {
		effectiveChunkSize = t.config.ChunkSize
	}

	if t.config.Verbose {
		fmt.Printf("Using effective chunk size of %d tokens (original: %d)\n",
			effectiveChunkSize, t.config.ChunkSize)
	}

	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	currentChunk := ""
	currentTokens := 0

	for _, paragraph := range paragraphs {

		paragraphTokens := len(paragraph) / 4

		if paragraphTokens > effectiveChunkSize {
			if currentChunk != "" {
				chunks = append(chunks, currentChunk)
				currentChunk = ""
				currentTokens = 0
			}

			lines := strings.Split(paragraph, "\n")

			if len(lines) > 1 {
				for _, line := range lines {
					lineTokens := len(line) / 4

					if currentTokens > 0 && (currentTokens+lineTokens+1) > effectiveChunkSize {
						chunks = append(chunks, currentChunk)
						currentChunk = line
						currentTokens = lineTokens
					} else {
						if currentTokens > 0 {
							currentChunk += "\n"
							currentTokens += 1
						}
						currentChunk += line
						currentTokens += lineTokens
					}
				}
			} else {

				sentenceSplitters := []string{". ", "! ", "? ", "; "}
				text := paragraph
				var sentences []string

				for _, splitter := range sentenceSplitters {
					parts := strings.Split(text, splitter)
					if len(parts) > 1 {
						for i := 0; i < len(parts)-1; i++ {
							sentences = append(sentences, parts[i]+splitter[:1])
						}
						text = parts[len(parts)-1]
					}
				}

				if text != "" {
					sentences = append(sentences, text)
				}

				if len(sentences) <= 1 {
					for i := 0; i < len(paragraph); i += effectiveChunkSize * 4 {
						end := i + effectiveChunkSize*4
						if end > len(paragraph) {
							end = len(paragraph)
						}
						chunks = append(chunks, paragraph[i:end])
					}
				} else {

					for _, sentence := range sentences {
						sentenceTokens := len(sentence) / 4

						if currentTokens > 0 && (currentTokens+sentenceTokens) > effectiveChunkSize {
							chunks = append(chunks, currentChunk)
							currentChunk = sentence
							currentTokens = sentenceTokens
						} else {
							currentChunk += sentence
							currentTokens += sentenceTokens
						}
					}
				}
			}
		} else {

			if currentTokens > 0 && (currentTokens+paragraphTokens+1) > effectiveChunkSize {

				chunks = append(chunks, currentChunk)
				currentChunk = paragraph
				currentTokens = paragraphTokens
			} else {

				if currentTokens > 0 {

					currentChunk += "\n\n"
					currentTokens += 1
				}
				currentChunk += paragraph
				currentTokens += paragraphTokens
			}
		}

		if currentTokens >= effectiveChunkSize {
			chunks = append(chunks, currentChunk)
			currentChunk = ""
			currentTokens = 0
		}
	}

	if currentChunk != "" {
		chunks = append(chunks, currentChunk)
	}

	if t.config.Verbose {
		for i, chunk := range chunks {
			fmt.Printf("Chunk %d: ~%d tokens (%d characters)\n",
				i+1, len(chunk)/4, len(chunk))
		}
	}

	return chunks
}

type OpenRouterRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenRouterResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (t *Translator) translateChunk(text string) (string, error) {

	prompt := fmt.Sprintf("Translate the following text to %s language, but save formatting, the answer place in the tag <result>:\n\n%s",
		t.config.ToLang, text)

	request := OpenRouterRequest{
		Model: t.config.Model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.config.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/hightemp/go_ai_translate")
	req.Header.Set("X-Title", "Go AI Translate")

	client := &http.Client{Timeout: 5 * time.Minute}

	var resp *http.Response
	maxRetries := t.config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	retryDelay := 2 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			if t.config.Verbose {
				fmt.Printf("Retrying API call (attempt %d/%d) after error: %v\n",
					attempt+1, maxRetries, err)
			}
			time.Sleep(retryDelay)

			retryDelay *= 2
		}

		resp, err = client.Do(req)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		errorMsg := fmt.Sprintf("API request failed with status %d: %s", resp.StatusCode, string(body))

		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Error.Message != "" {
			errorMsg = fmt.Sprintf("API request failed: %s (Type: %s, Code: %s)",
				errorResponse.Error.Message,
				errorResponse.Error.Type,
				errorResponse.Error.Code)
		}

		return "", fmt.Errorf("%s", errorMsg)
	}

	var response OpenRouterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if response.Error != nil {
		errorMsg := fmt.Sprintf("API error: %s", response.Error.Message)

		if t.config.Verbose {
			fmt.Printf("API error details: %s\n", errorMsg)
			fmt.Printf("Request body: %s\n", string(requestBody))
		}

		return "", fmt.Errorf("%s", errorMsg)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no translation returned from API")
	}

	translation := response.Choices[0].Message.Content

	result, err := t.extractResultTag(translation)

	if err != nil {
		return "", err
	}

	return result, nil
}

func (t *Translator) extractResultTag(input string) (string, error) {
	re := regexp.MustCompile(`(?s)<result>(.*?)</result>`)

	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return "", fmt.Errorf("tag <result> not found")
	}

	return matches[1], nil
}
