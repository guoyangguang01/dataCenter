package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type StreamConfig struct {
	Name      string        `yaml:"name"`
	Subjects  []string      `yaml:"subjects"`
	Retention string        `yaml:"retention"`
	MaxAge    time.Duration `yaml:"max_age"`
	Storage   string        `yaml:"storage"`
}

var DefaultStreams = []StreamConfig{
	{
		Name:      "DEVICE_DATA",
		Subjects:  []string{"domains.>.devices.>.>.>.up"},
		Retention: "limits",
		MaxAge:    7 * 24 * time.Hour,
		Storage:   "file",
	},
	{
		Name:      "DEVICE_COMMAND",
		Subjects:  []string{"domains.>.devices.>.>.>.down"},
		Retention: "workqueue",
		MaxAge:    3 * 24 * time.Hour,
		Storage:   "file",
	},
	{
		Name:      "SYSTEM_EVENTS",
		Subjects:  []string{"system.events", "system.>"},
		Retention: "limits",
		MaxAge:    30 * 24 * time.Hour,
		Storage:   "file",
	},
}

func EnsureStream(ctx context.Context, js jetstream.JetStream, config StreamConfig) error {
	retention := jetstream.LimitsPolicy
	if config.Retention == "workqueue" {
		retention = jetstream.WorkQueuePolicy
	}
	storage := jetstream.FileStorage
	if config.Storage == "memory" {
		storage = jetstream.MemoryStorage
	}
	streamConfig := jetstream.StreamConfig{
		Name:      config.Name,
		Subjects:  config.Subjects,
		Retention: retention,
		MaxAge:    config.MaxAge,
		Storage:   storage,
	}
	_, err := js.CreateOrUpdateStream(ctx, streamConfig)
	if err != nil {
		return fmt.Errorf("failed to create stream %s: %w", config.Name, err)
	}
	fmt.Println("[NATS] stream ensured:", config.Name)
	return nil
}

func EnsureAllStreams(ctx context.Context, js jetstream.JetStream) error {
	for _, config := range DefaultStreams {
		if err := EnsureStream(ctx, js, config); err != nil {
			return err
		}
	}
	return nil
}
