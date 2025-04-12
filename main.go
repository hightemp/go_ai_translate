package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hightemp/go_ai_translate/translator"
)

func main() {
	inputFile := flag.String("input", "", "Input file to translate (required)")
	outputFile := flag.String("output", "", "Output file for translation (required)")
	toLang := flag.String("to", "russian", "Target language (default: russian)")
	apiKey := flag.String("api-key", os.Getenv("OPENROUTER_API_KEY"), "OpenRouter API key (default from env OPENROUTER_API_KEY)")
	chunkSize := flag.Int("chunk-size", 500, "Size of text chunks in tokens (default: 500)")
	model := flag.String("model", "deepseek/deepseek-chat", "Model to use for translation (default: deepseek/deepseek-chat)")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	maxRetries := flag.Int("max-retries", 3, "Maximum number of retries for API calls (default: 3)")

	flag.Parse()

	if *inputFile == "" || *outputFile == "" || *apiKey == "" {
		fmt.Println("Error: input file, output file, and API key are required")
		flag.Usage()
		os.Exit(1)
	}

	config := translator.Config{
		APIKey:     *apiKey,
		ToLang:     *toLang,
		ChunkSize:  *chunkSize,
		Model:      *model,
		Verbose:    *verbose,
		MaxRetries: *maxRetries,
	}

	if *verbose {
		fmt.Printf("Configuration:\n")
		fmt.Printf("  To language: %s\n", *toLang)
		fmt.Printf("  Chunk size: %d tokens\n", *chunkSize)
		fmt.Printf("  Model: %s\n", *model)
		fmt.Printf("  Max retries: %d\n", *maxRetries)
	}

	t := translator.NewTranslator(config)

	outputDir := filepath.Dir(*outputFile)
	if outputDir != "" && outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("Error creating output directory: %v\n", err)
			os.Exit(1)
		}
	}

	if *verbose {
		fmt.Printf("Starting translation of %s to %s...\n", *inputFile, *outputFile)
	}

	startTime := time.Now()
	if err := t.TranslateFile(*inputFile, *outputFile); err != nil {
		fmt.Printf("Error translating file: %v\n", err)
		os.Exit(1)
	}

	elapsedTime := time.Since(startTime)
	fmt.Printf("Translation completed successfully in %v. Output written to %s\n",
		elapsedTime.Round(time.Second), *outputFile)
}
