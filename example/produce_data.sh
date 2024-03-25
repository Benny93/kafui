#!/bin/bash

# Function to fetch and send exchange rate data to Kafka
fetch_and_send_data() {
    # Fetch data from the API
    response=$(curl -s "https://api.coinbase.com/v2/exchange-rates?currency=BTC")

    if [ -n "$response" ]; then
        # Pipe the exchange rate to Kafka
        echo "$response" | kaf produce Bitcoin --key BTC
    else
        echo "Error: Failed to fetch exchange rate for BTC."
    fi
}

# Default number of times to run the script
count=1

# Check if the user specified a different count
if [ $# -eq 1 ]; then
    count=$1
fi

# Loop to run the script 'count' number of times
for ((i=0; i<count; i++)); do
    fetch_and_send_data
done
