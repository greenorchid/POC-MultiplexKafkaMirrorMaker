/**
 * Copyright 2016 Confluent Inc.
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

// Example function-based Apache Kafka producer
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

func main() {

	var bootstrapServers, demuxTopic string
	var group int
	var sourceTopics []string

	// Define the command-line options
	flag.StringVar(&bootstrapServers, "bootstrap", "localhost:9092", "Bootstrap server")
	flag.IntVar(&group, "group", 0, "Consumer Group")
	flag.StringVar(&demuxTopic, "demux-topics", "mux-replication_siteA mux-replication_siteB", "source topic(s) to produce demultiplexed messages")

	// Parse the command-line options
	flag.Parse()

	sourceTopics = append(sourceTopics, demuxTopic)
	// append any additional args as sourceTopics
	sourceTopics = append(sourceTopics, flag.Args()...)

	// Parse the command-line options
	flag.Parse()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":     bootstrapServers,
		"broker.address.family": "v4",
		"group.id":              group,
		"session.timeout.ms":    6000,
		// Start reading from the first message of each assigned
		// partition if there are no previously committed offsets
		// for this group.
		"auto.offset.reset": "earliest",
		// Whether or not we store offsets automatically.
		"enable.auto.offset.store": false,
	})

	if err != nil {
		fmt.Printf("Failed to create consumer: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created Consumer %v\n", c)

	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": bootstrapServers})
	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created Producer %v\n", p)

	// Listen to all the events on the default events channel
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				// The message delivery report, indicating success or
				// permanent failure after retries have been exhausted.
				// Application level retries won't help since the client
				// is already configured to do that.
				m := ev
				if m.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
				} else {
					fmt.Printf("Delivered message to topic \"%s [%d]\" at offset %v\n",
						*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
				}
			case kafka.Error:
				// Generic client instance-level errors, such as
				// broker connection failures, authentication issues, etc.
				//
				// These errors should generally be considered informational
				// as the underlying client will automatically try to
				// recover from any errors encountered, the application
				// does not need to take action on them.
				fmt.Printf("Error: %v\n", ev)
			default:
				fmt.Printf("Ignored event: %s\n", ev)
			}
		}
	}()

	err = c.SubscribeTopics(sourceTopics, nil)

	if err != nil {
		fmt.Printf("Consumer failed to subscribe to topics: %s\n", err)
		os.Exit(1)
	}

	run := true

	for run {
		select {
		case sig := <-sigchan:
			fmt.Printf("Caught signal %v: terminating\n", sig)
			run = false
		default:
			ev := c.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				// Process the message received.

				fmt.Printf("%% Message on %s:\n%s\n",
					e.TopicPartition, string(e.Value))

				var demuxTopic string
				var demuxTopicPartition int32

				// iterate through the headers
				for _, header := range e.Headers {
					// Check each header's key
					switch string(header.Key) {
					case "MUX_SOURCE_TOPIC":
						demuxTopic = string(header.Value)

					case "MUX_SOURCE_TOPIC_PARTITION":
						i, _ := strconv.Atoi(string(header.Value))
						demuxTopicPartition = int32(i)
					}

				}

				err = p.Produce(&kafka.Message{
					TopicPartition: kafka.TopicPartition{
						Topic:     &demuxTopic,
						Partition: demuxTopicPartition,
					},
					Value:   []byte(e.Value),
					Key:     e.Key,
					Headers: e.Headers,
				}, nil)

				if err != nil {
					if err.(kafka.Error).Code() == kafka.ErrQueueFull {
						// Producer queue is full, wait 1s for messages
						// to be delivered then try again.
						time.Sleep(time.Second)
						continue
					}
					fmt.Printf("Failed to produce message: %v\n", err)
				}

				_, err := c.StoreMessage(e)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%% Error storing offset after message %s:\n",
						e.TopicPartition)
				}
			case kafka.Error:
				// Errors should generally be considered
				// informational, the client will try to
				// automatically recover.
				// But in this example we choose to terminate
				// the application if all brokers are down.
				fmt.Fprintf(os.Stderr, "%% Error: %v: %v\n", e.Code(), e)
				if e.Code() == kafka.ErrAllBrokersDown {
					run = false
				}
			default:
				fmt.Printf("Ignored %v\n", e)
			}
		}
	}

	fmt.Printf("Closing consumer\n")
	c.Close() //TODO I think I should defer() this

}
