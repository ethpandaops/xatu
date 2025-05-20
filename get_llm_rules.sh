#!/bin/bash

# get_llm_rules.sh
# A script to scan and collect rule definitions from llm rule files.
# This script returns rule content as standard output so it can be used directly by LLMs.
#
# Usage:
#   ./get_llm_rules.sh [context_filter]
#
# Parameters:
#   context_filter - Optional: Only return rules matching this context (e.g., server, sentry, ethereum)
#
# Examples:
#   ./get_llm_rules.sh               # Return all rules
#   ./get_llm_rules.sh server        # Return only server rules
#   ./get_llm_rules.sh ethereum      # Return only ethereum-related rules

# Set variables
RULES_DIR=".cursor/rules"
CONTEXT_FILTER="$1"

# Function to check if a rule file matches a context filter
matches_context() {
  local file="$1"
  local filter="$2"
  
  # If no filter specified, include all files
  if [ -z "$filter" ]; then
    return 0
  fi
  
  # Extract metadata from the file
  local description=$(grep -A2 "^---" "$file" | grep "description:" | sed 's/description: *//')
  local globs=$(grep -A3 "^---" "$file" | grep "globs:" | sed 's/globs: *//')
  
  # Check if the filter appears in:
  # 1. The rule filename
  # 2. The description
  # 3. The glob patterns
  if [[ "$(basename "$file")" == *"$filter"* ]] || 
     [[ "$description" == *"$filter"* ]] || 
     [[ "$globs" == *"$filter"* ]]; then
    return 0
  fi
  
  # Also check the content of the file for keywords
  if grep -q -i "$filter" "$file"; then
    return 0
  fi
  
  return 1
}

# Function to extract and format rule content
format_rule() {
  local file="$1"
  local name=$(basename "$file" .mdc)
  local content=$(sed -n '/^---/,/^---/!p' "$file" | sed '1d')
  
  # Extract metadata
  local description=$(grep -A2 "^---" "$file" | grep "description:" | sed 's/description: *//')
  local globs=$(grep -A3 "^---" "$file" | grep "globs:" | sed 's/globs: *//')
  
  echo "===========================================================" 
  echo "RULE: $name"
  echo "DESCRIPTION: $description"
  echo "APPLIES TO: $globs"
  echo "===========================================================" 
  echo "$content"
  echo ""
  echo ""
}

# Main execution
if [ ! -d "$RULES_DIR" ]; then
  echo "Error: Rules directory not found at $RULES_DIR" >&2
  exit 1
fi

# Find all rule files
rule_files=$(find "$RULES_DIR" -name "*.mdc" 2>/dev/null)

if [ -z "$rule_files" ]; then
  echo "No rule files found in $RULES_DIR" >&2
  exit 1
fi

echo "# LLM Rules Reference"
echo ""
echo "The following rules are relevant to your current task:"
echo ""

for file in $rule_files; do
  if matches_context "$file" "$CONTEXT_FILTER"; then
    format_rule "$file"
  fi
done

echo "END OF RULES REFERENCE"