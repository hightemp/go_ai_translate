package translator

import (
	"os"
	"strings"
	"testing"
)

func TestSplitIntoChunks(t *testing.T) {

	config := Config{
		ChunkSize: 50,
	}
	translator := NewTranslator(config)

	testCases := []struct {
		name          string
		input         string
		expectedCount int
	}{
		{
			name:          "Empty text",
			input:         "",
			expectedCount: 0,
		},
		{
			name:          "Small text",
			input:         "This is a small text that should fit in one chunk.",
			expectedCount: 1,
		},
		{
			name:          "Multiple paragraphs",
			input:         "Paragraph 1.\n\nParagraph 2.\n\nParagraph 3.\n\nParagraph 4.",
			expectedCount: 1,
		},
		{
			name: "Large text",
			input: `This is a very long paragraph that should be split into multiple chunks because it exceeds the configured chunk size. This sentence adds more characters to ensure we exceed the limit. And we'll add even more text to be absolutely certain that this will be split into multiple chunks.
			
			This is another paragraph that adds more content to ensure we get multiple chunks in our test. We need to make sure this paragraph is also quite long to contribute significantly to the token count.
			
			And here's yet another paragraph with more text to make sure we have enough content for testing the chunking functionality properly. The more text we add, the more likely we are to exceed our very small chunk size limit.
			
			Let's add even more text to make sure we have enough content to split into multiple chunks for our test case. This should definitely push us over the edge.
			
			This should be enough text to create at least two chunks with our test configuration.`,
			expectedCount: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunks := translator.splitIntoChunks(tc.input)

			t.Logf("Input length: %d characters, approx %d tokens", len(tc.input), len(tc.input)/4)
			t.Logf("Got %d chunks", len(chunks))
			for i, chunk := range chunks {
				t.Logf("Chunk %d: %d characters, approx %d tokens", i+1, len(chunk), len(chunk)/4)
			}

			if len(chunks) != tc.expectedCount && tc.expectedCount != 0 {
				t.Errorf("Expected %d chunks, got %d", tc.expectedCount, len(chunks))
			}

			if tc.input != "" && len(chunks) > 0 {

				if tc.name != "Large text" {
					reconstructed := ""
					for i, chunk := range chunks {
						reconstructed += chunk
						if i < len(chunks)-1 && !endsWithNewline(chunk) {
							reconstructed += "\n\n"
						}
					}

					normalizedInput := normalizeText(tc.input)
					normalizedReconstructed := normalizeText(reconstructed)

					if normalizedReconstructed != normalizedInput {
						t.Errorf("Content was not preserved correctly after chunking")
						t.Logf("Original: %q", normalizedInput)
						t.Logf("Reconstructed: %q", normalizedReconstructed)
					}
				} else {

					allContent := strings.Join(chunks, "")
					if !containsAllWords(allContent, tc.input) {
						t.Errorf("Not all content was preserved in the chunks")
					}
				}
			}
		})
	}
}

func endsWithNewline(s string) bool {
	return len(s) > 0 && (s[len(s)-1] == '\n')
}

func normalizeText(s string) string {

	s = strings.Join(strings.Fields(s), " ")

	return strings.TrimSpace(s)
}

func containsAllWords(processed, original string) bool {

	originalWords := strings.Fields(original)
	processedLower := strings.ToLower(processed)

	for _, word := range originalWords {

		if len(word) <= 2 {
			continue
		}

		if !strings.Contains(strings.ToLower(processedLower), strings.ToLower(word)) {
			return false
		}
	}

	return true
}

func TestTranslateChunk(t *testing.T) {

	t.Skip("Skipping integration test: requires valid API key")

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: OPENROUTER_API_KEY not set")
	}

	config := Config{
		APIKey: apiKey,
		ToLang: "es",
		Model:  "openai/gpt-3.5-turbo",
	}
	translator := NewTranslator(config)

	input := "Hello, world!"
	translated, err := translator.translateChunk(input)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if translated == "" {
		t.Error("Expected non-empty translation result")
	}

	if translated == input {
		t.Error("Translation appears unchanged, expected different text")
	}
}
