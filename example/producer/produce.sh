#!/bin/bash
echo "Producing avro message to localhost ..."
go run produce_avro.go user.go localhost:9092 http://localhost:8085 first.messages