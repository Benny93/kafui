// Example function-based Apache Kafka producer
package main

/**
 * Copyright 2022 Confluent Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import (
	"fmt"
	"os"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
)

func main() {
	if len(os.Args) < 4 || len(os.Args) > 5 {
		fmt.Fprintf(os.Stderr, "Usage: %s <bootstrap-servers> <schema-registry> <topic> [count]\n",
			os.Args[0])
		fmt.Fprintf(os.Stderr, "  bootstrap-servers: e.g., 127.0.0.1:9092 or 127.0.0.1:9092,127.0.0.1:9093\n")
		fmt.Fprintf(os.Stderr, "  schema-registry:   e.g., http://127.0.0.1:8081\n")
		fmt.Fprintf(os.Stderr, "  topic:             e.g., test.users\n")
		fmt.Fprintf(os.Stderr, "  count:             number of messages to produce (default: 100)\n")
		os.Exit(1)
	}

	bootstrapServers := os.Args[1]
	url := os.Args[2]
	topic := os.Args[3]
	numUsers := 100 // Default number of messages

	if len(os.Args) == 5 {
		if _, err := fmt.Sscanf(os.Args[4], "%d", &numUsers); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid count: %s\n", os.Args[4])
			os.Exit(1)
		}
	}

	fmt.Printf("📤 Producing %d messages to topic '%s'...\n", numUsers, topic)
	fmt.Printf("   Bootstrap servers: %s\n", bootstrapServers)
	fmt.Printf("   Schema Registry:   %s\n", url)
	fmt.Println()

	// Create shared producer
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":       bootstrapServers,
		"broker.address.family":   "v4",
		"message.timeout.ms":      10000,
		"delivery.timeout.ms":     20000,
	})
	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}
	defer p.Close()

	// Create shared schema registry client
	client, err := schemaregistry.NewClient(schemaregistry.NewConfig(url))
	if err != nil {
		fmt.Printf("Failed to create schema registry client: %s\n", err)
		os.Exit(1)
	}

	ser, err := avro.NewSpecificSerializer(client, serde.ValueSerde, avro.NewSerializerConfig())
	if err != nil {
		fmt.Printf("Failed to create serializer: %s\n", err)
		os.Exit(1)
	}

	successCount := 0

	// Produce messages
	for i := 0; i < numUsers; i++ {
		value := User{
			Name:            fmt.Sprintf("User %d", i),
			Favorite_number: int64(i),
			Favorite_color:  "blue",
		}
		payload, err := ser.Serialize(topic, &value)
		if err != nil {
			fmt.Printf("Failed to serialize payload: %s\n", err)
			continue
		}

		// Produce message
		err = p.Produce(&kafka.Message{
			Key:            []byte(fmt.Sprintf("user-%d", i)),
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          payload,
			Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
		}, nil)
		if err != nil {
			fmt.Printf("Produce failed: %v\n", err)
			continue
		}
		successCount++
	}

	// Flush to ensure all messages are delivered
	remaining := p.Flush(30000)
	if remaining > 0 {
		fmt.Printf("⚠️  Warning: %d messages still in queue after flush\n", remaining)
	}

	fmt.Printf("✅ Successfully produced %d/%d messages to topic '%s'\n", successCount, numUsers, topic)
}
