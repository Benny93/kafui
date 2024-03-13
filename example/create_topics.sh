#!/bin/bash

# Function to create Kafka topics
create_topic() {
    local topic_name="$1"
    local num_partitions="$2"
    local replication_factor="$3"
    
    #kafka-topics.sh --create \
    #                --zookeeper localhost:2181 \
    #                --topic "$topic_name" \
    #                --partitions "$num_partitions" \
    #                --replication-factor "$replication_factor"
    kaf topic create "$topic_name" -p "$num_partitions"
}

# Cryptocurrency names
cryptocurrencies=("Bitcoin" "Ethereum" "Ripple" "Litecoin" "Bitcoin Cash" "Cardano" "Polkadot" "Chainlink" "Stellar" "Dogecoin" "Solana")

# Define number of iterations for creating topics
num_iterations=30

# Loop through iterations and create topics
for ((i=1; i<=$num_iterations; i++)); do
    # Choose a random cryptocurrency name
    random_index=$((RANDOM % ${#cryptocurrencies[@]}))
    topic_name="${cryptocurrencies[$random_index]}_topic_$i"
    
    # Random number of partitions (1 to 5)
    num_partitions=$((RANDOM % 5 + 1))
    
    # Random replication factor (1 to 3)
    replication_factor=$((RANDOM % 3 + 1))
    
    echo "Creating topic: $topic_name with $num_partitions partitions and replication factor $replication_factor"
    create_topic "$topic_name" "$num_partitions" "$replication_factor"
done
