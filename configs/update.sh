#!/bin/bash

# GitHub repository and raw base URL
REPO_URL="https://github.com/lra/mackup/tree/master/mackup/applications"
RAW_BASE_URL="https://raw.githubusercontent.com/lra/mackup/master/mackup/applications"

echo "Fetching file list from $REPO_URL..."

# Fetch the GitHub repository page and extract file names
FILE_URLS=$(curl -s "$REPO_URL" | grep 'href="/lra/mackup/blob/master/mackup/applications/' | \
    sed -E 's|.*href="/lra/mackup/blob/master/mackup/applications/([^"]+)".*|\1|' | \
    awk -v base="$RAW_BASE_URL" '{print base "/" $1}')

if [ -z "$FILE_URLS" ]; then
    echo "No files found to download. Exiting."
    exit 1
fi

echo "Downloading files..."
for FILE_URL in $FILE_URLS; do
    FILE_NAME=$(basename "$FILE_URL")
    echo "Downloading $FILE_NAME..."
    curl -s -O "$FILE_URL"
    if [ $? -eq 0 ]; then
        echo "Downloaded: $FILE_NAME"
    else
        echo "Failed to download: $FILE_NAME"
    fi
done

echo "All files downloaded."