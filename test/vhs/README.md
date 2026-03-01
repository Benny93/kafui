# VHS Integration Tests for Kafui

This directory contains VHS integration tests for the Kafui Kafka UI application.

## What is VHS?

VHS is a tool for creating terminal GIFs and testing terminal applications.
It allows you to write test scenarios in a simple tape format and replay them
to verify application behavior.

See: https://github.com/charmbracelet/vhs

## Prerequisites

1. Install VHS:
   go install github.com/charmbracelet/vhs@latest

2. Docker (optional, for running Kafka locally)

## Quick Start

Run topic navigation test with mock data:
   go test ./test/vhs/... -run TestVHS_TopicNavigation -v

## Test Scenarios

### topic_navigation_mock.tape

Tests basic topic navigation using mock data.
Duration: ~30 seconds

### topic_navigation.tape

Tests topic navigation with real Kafka data.
Duration: ~45 seconds
Requirements: Running Kafka instance on localhost:9092

## Running Tests

All VHS Tests:
   go test ./test/vhs/... -v

Individual Tests:
   go test ./test/vhs/... -run TestVHS_TopicNavigation -v
   go test ./test/vhs/... -run TestVHS_ValidateTapes -v

Using VHS Directly:
   vhs test/vhs/tapes/topic_navigation_mock.tape
   vhs --validate test/vhs/tapes/topic_navigation_mock.tape

## Keyboard Shortcuts Tested

Navigation:
- Up/Down arrows - Navigate lists
- j/k - Vim-style navigation  
- Home/End - Jump to top/bottom
- PageUp/PageDown - Page navigation

Actions:
- Enter - Select/open item
- Escape - Go back
- q - Quit application
- r - Refresh data

## Troubleshooting

VHS not installed:
   go install github.com/charmbracelet/vhs@latest

Kafka connection failed:
   docker-compose -f test/docker/docker-compose.yml up -d

## Resources

- VHS Documentation: https://github.com/charmbracelet/vhs
- VHS Examples: https://github.com/charmbracelet/vhs/tree/main/examples
