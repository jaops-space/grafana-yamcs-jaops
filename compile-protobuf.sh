#!/bin/bash

# Yamcs Protobuf Compilation Script
# This script clones the latest Yamcs repository, extracts the protobuf files,
# updates them with Go package options, compiles them, and cleans up.

set -e  # Exit immediately if a command fails

# Colors
GREEN="\e[32m"
YELLOW="\e[33m"
CYAN="\e[36m"
PURPLE="\e[0;35m"
BOLD="\e[1m"
RESET="\e[0m"
CLEAR_LINE="\033[A\e[2K\r"

REPO_URL="https://github.com/yamcs/yamcs.git"
TEMP_REPO_DIR="temp-yamcs"
PROTO_SRC_DIR="$TEMP_REPO_DIR/yamcs-api/src/main/proto/yamcs"
PROTO_DEST_DIR="./api/yamcs"
BASE_API_DIR="./api"
GO_PKG_BASE="github.com/jaops-space/grafana-yamcs-jaops/api/yamcs"

CURRENT=0
TOTAL=0

progress_bar() {
  local file=$1
  CURRENT=$((CURRENT + 1))
  local percent=$((CURRENT * 100 / TOTAL))
  local completed=$((percent / 5))
  printf "\e[2K${CYAN}[%s%s] %d%% ${PURPLE}${file}${RESET}\r" \
    "$(printf '█%.0s' $(seq 1 $completed))" "$(printf -- " %.0s" $(seq $completed 19))" "$percent"
}

progress_bar_end() {
  printf "\n${CLEAR_LINE}"
}

# Step 1: Check if temp-yamcs exists, if not clone
printf "${CYAN}>>> Checking for existing Yamcs repository...${RESET}\n"
if [ -d "$TEMP_REPO_DIR" ]; then
  printf "${CLEAR_LINE}${YELLOW}✔ Yamcs repository already exists, skipping clone.${RESET}\n"
else
  printf "${CYAN}>>> Cloning the latest Yamcs repository...${RESET}\n"
  git clone --depth=1 $REPO_URL $TEMP_REPO_DIR
  printf "${CLEAR_LINE}${GREEN}✔ Cloning the latest Yamcs repository... done${RESET}\n"
fi

# Step 2: Clear previous protobuf files
printf "${CYAN}>>> Removing old protobuf definitions...${RESET}\n"
rm -rf $PROTO_DEST_DIR
mkdir -p $PROTO_DEST_DIR
printf "${CLEAR_LINE}${GREEN}✔ Removing old protobuf definitions... done${RESET}\n"

# Step 3: Copy new protobuf files
printf "${CYAN}>>> Copying fresh protobuf files from Yamcs repo...${RESET}\n"
TOTAL=$(find $PROTO_SRC_DIR -name '*.proto' | wc -l)
find "$PROTO_SRC_DIR" -name '*.proto' | while read -r proto_file; do
    relative_file=$(realpath --relative-to="$PROTO_SRC_DIR" "$proto_file") 
    mkdir -p "$(dirname "$PROTO_DEST_DIR/$relative_file")"  # Ensure directories exist
    cp "$proto_file" "$PROTO_DEST_DIR/$relative_file"  # Copy while keeping structure
    progress_bar "$relative_file"
done
progress_bar_end
printf "${CLEAR_LINE}${GREEN}✔ Copying fresh protobuf files from Yamcs repo... done${RESET}\n"

# Step 4: Add 'option go_package' to .proto files if missing
printf "${CYAN}>>> Updating Go package options in protobuf files...${RESET}\n"
TOTAL=$(find $PROTO_DEST_DIR -name '*.proto' | wc -l)
find $PROTO_DEST_DIR -name '*.proto' | while read -r proto_file; do
    relative_dir=$(dirname "$(realpath --relative-to="$PROTO_DEST_DIR" "$proto_file")")
    go_package="$GO_PKG_BASE/$relative_dir"
    if ! grep -q 'option go_package' "$proto_file"; then
        echo "option go_package = \"$go_package\";" >> "$proto_file"
    fi
  progress_bar "$proto_file"
done
progress_bar_end
printf "${CLEAR_LINE}${GREEN}✔ Updating Go package options in protobuf files... done${RESET}\n"

# Step 5: Compile .proto files
printf "${CYAN}>>> Compiling protobuf files...${RESET}\n"
TOTAL=$(find $PROTO_DEST_DIR -name '*.proto' | wc -l)
find $PROTO_DEST_DIR -name '*.proto' | while read -r proto_file; do
    protoc -I=$BASE_API_DIR --go_out=./ $proto_file
    progress_bar "$proto_file"
done
progress_bar_end
printf "${CLEAR_LINE}${GREEN}✔ Compiling protobuf files... done${RESET}\n"

# Step 6: Cleanup compiled files
printf "${CYAN}>>> Organizing compiled files...${RESET}\n"
rm -rf $PROTO_DEST_DIR
mkdir -p $PROTO_DEST_DIR
mv ./$GO_PKG_BASE/* $PROTO_DEST_DIR
printf "${CLEAR_LINE}${GREEN}✔ Organizing compiled files... done${RESET}\n"

# Step 7: Remove original .proto files
printf "${CYAN}>>> Cleaning up...${RESET}\n"
find $PROTO_DEST_DIR -name '*.proto' -exec rm {} \;
rm -rf ./github.com/
printf "${CLEAR_LINE}${GREEN}✔ Cleaning up... done${RESET}\n"

printf "\n${BOLD}${GREEN}Done! Protobuf Go files compilation finished successfully.${RESET}\n"
exit 0
