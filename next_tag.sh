#!/bin/bash

# Get the last tag
last_tag=$(git describe --abbrev=0 --tags)

# Parse the last tag
IFS='.' read -r -a parts <<< "$last_tag"
major="${parts[0]}"
minor="${parts[1]}"
patch="${parts[2]}"

# Increment the patch version
next_patch=$((patch + 1))

# Generate the next tag
next_tag="$major.$minor.$next_patch"
echo "$next_tag"
