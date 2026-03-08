package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/ifsix/GoRouter-kit"
	"github.com/ifsix/GoRouter-kit/history/memory"
)

func main() {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		log.Fatal("OPENROUTER_API_KEY is required")
	}

	cli, err := gorouter.New(gorouter.Config{
		APIKey:      key,
		AppName:     "GoRouterKit Example",
		HTTPReferer: "https://example.com",
		Model:       "openai/gpt-4o-mini",
		History:     memory.New(),
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := cli.Chat(context.Background(), gorouter.ChatRequest{
		SessionID: "demo",
		Provider: &gorouter.Provider{
			Order: []string{"openai"},
		},
		Messages: []gorouter.Message{
			{Role: gorouter.RoleUser, Content: "Say hello in one short sentence."},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	if len(resp.Choices) == 0 {
		log.Fatal("empty response")
	}

	fmt.Println(resp.Choices[0].Message.Content)
}
