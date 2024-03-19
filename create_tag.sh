#!/bin/bash

# Check if tag version and message are provided
if [ $# -ne 2 ]; then
    echo "Usage: $0 <tag_version> <tag_message>"
    exit 1
fi

# Extract parameters
TAG_VERSION="$1"
TAG_MESSAGE="$2"

# Create the new tag
git tag -a "$TAG_VERSION" -m "$TAG_MESSAGE"

# Push the new tag to the remote repository
git push origin "$TAG_VERSION"

echo "Tag $TAG_VERSION created and pushed successfully."
