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
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde"
	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <bootstrap-servers> <schema-registry> <topic>\n",
			os.Args[0])
		os.Exit(1)
	}

	bootstrapServers := os.Args[1]
	url := os.Args[2]
	topic := os.Args[3]

	numUsers := 100 // Number of users to produce
	var wg sync.WaitGroup
	wg.Add(numUsers)

	// Create producers concurrently
	for i := 0; i < numUsers; i++ {
		go func(i int) {
			defer wg.Done()

			p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": bootstrapServers})
			if err != nil {
				fmt.Printf("Failed to create producer: %s\n", err)
				return
			}

			client, err := schemaregistry.NewClient(schemaregistry.NewConfig(url))
			if err != nil {
				fmt.Printf("Failed to create schema registry client: %s\n", err)
				return
			}

			ser, err := avro.NewSpecificSerializer(client, serde.ValueSerde, avro.NewSerializerConfig())
			if err != nil {
				fmt.Printf("Failed to create serializer: %s\n", err)
				return
			}

			defer p.Close()

			value := User{
				Name:            fmt.Sprintf("User %d", i),
				Favorite_number: int64(i),
				Favorite_color:  "blue",
			}
			payload, err := ser.Serialize(topic, &value)
			if err != nil {
				fmt.Printf("Failed to serialize payload: %s\n", err)
				return
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
				return
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("All users produced successfully")
}
