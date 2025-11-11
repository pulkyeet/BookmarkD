package analytics

import (
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/posthog/posthog-go"
)

var client posthog.Client

func InitPosthog() error {
	apiKey := os.Getenv("POSTHOG_API_KEY")
	host := os.Getenv("POSTHOG_HOST")
	
	if apiKey == "" || host == "" {
		return fmt.Errorf("Posthog API Key or Host missing")
	}
	var err error
	client, err = posthog.NewWithConfig(apiKey, posthog.Config{
		Endpoint: host,
	},)
	if err != nil {
		return fmt.Errorf("Failed to initialise Posthog: %w", err)
	}
	log.Println("Posthog initialised successfully.")
	return nil
}

func Close() {
	if client != nil {
		client.Close()
	}
}

func Track(userID, event string, properties map[string]interface{}) {
	if client == nil {
		return
	}
	if properties == nil {
		properties = make(map[string]interface{})
	}
	err := client.Enqueue(posthog.Capture{
		DistinctId: userID,
		Event: event,
		Properties: properties,
	})
	if err != nil {
		log.Printf("Posthog tracking error: %v", err)
	}
}

func Identify(userID string, properties map[string]interface{}) {
	if client == nil {
		return
	}
	err := client.Enqueue(posthog.Identify{
		DistinctId: userID,
		Properties: properties,
	})
	if err != nil {
		log.Printf("Posthog identify error: %v", err)
	}
}

func PageView(userID, path string, properties map[string]interface{}) {
	if properties == nil {
		properties = make(map[string]interface{})
	}
	properties["$current_url"] = path
	Track(userID, "$pageview", properties)
}