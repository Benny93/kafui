# Redpanda Development Cluster

A lightweight Redpanda cluster for developing and testing Kafka CLI tools like **kafui**.

Redpanda is a Kafka API-compatible streaming platform that includes an integrated Schema Registry, making it perfect for local development.

## Quick Start

### 1. Start the Cluster

```bash
cd example/redpanda-docker
./start.sh
```

This starts a single-node Redpanda cluster with:
- Kafka broker
- Integrated Schema Registry
- Pandaproxy (REST proxy)

### 2. Produce Test Messages

```bash
./produce.sh
```

Or with custom parameters:
```bash
./produce.sh <bootstrap-servers> <schema-registry> <topic> [count]

# Examples:
./produce.sh 127.0.0.1:9092 http://127.0.0.1:8081 test.users 50
./produce.sh 127.0.0.1:9092 http://127.0.0.1:8081 my.topic 1000
```

### 3. View Messages with kafui

```bash
# From project root
kafui --broker 127.0.0.1:9092
```

### 4. Stop the Cluster

```bash
./stop.sh
```

## Services

| Service | Endpoint | Description |
|---------|----------|-------------|
| **Kafka Broker** | `localhost:9092` | Main broker for clients |
| **Schema Registry** | `http://localhost:8081` | Avro schema registry |
| **Pandaproxy** | `http://localhost:8082` | REST proxy for Kafka |
| **Admin API** | `http://localhost:9644` | Redpanda admin API |

## Files

```
redpanda-docker/
├── docker-compose.yaml    # Cluster configuration
├── start.sh              # Start the cluster
├── stop.sh               # Stop the cluster
├── produce.sh            # Produce test messages
└── README.md             # This file

../producer/
├── produce_avro.go       # Avro producer
├── user.go               # Avro-generated User type
└── user.avsc             # Avro schema
```

## Using with kafui

### Connect to the cluster

```bash
kafui --broker 127.0.0.1:9092
```

### Configure Schema Registry

In kafui, configure the Schema Registry URL:
```
http://localhost:8081
```

### View Avro messages

Once connected, navigate to the topic page and kafui will automatically:
1. Fetch the schema from the registry
2. Deserialize Avro-encoded messages
3. Display them in readable JSON format

## Troubleshooting

### Check cluster status

```bash
docker compose ps
```

### View Redpanda logs

```bash
docker compose logs -f redpanda
```

### Reset the cluster

```bash
./stop.sh
docker volume rm redpanda-docker_redpanda-data
./start.sh
```

### Test connection with rpk

```bash
docker exec redpanda rpk cluster info
docker exec redpanda rpk topic list
docker exec redpanda rpk topic consume test.users -n 5
```

### Manual message production

```bash
# Using the Go producer
cd example/producer
go run produce_avro.go user.go 127.0.0.1:9092 http://127.0.0.1:8081 my.topic 100

# Using rpk (plain text)
docker exec redpanda rpk topic produce my.topic <<EOF
{"key": "test", "value": "hello world"}
EOF
```

## Requirements

- Docker & Docker Compose
- Go 1.21+ (for the producer example)
- 1GB+ free RAM
- 500MB+ free disk space

## Multi-node Cluster (Optional)

For testing multi-broker scenarios, you can modify `docker-compose.yaml` to add additional nodes. See the Redpanda documentation for multi-node configuration.
