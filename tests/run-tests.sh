#!/bin/bash

# Script to run go tests based on names listed in passed-tests.txt
# Usage: 
# 1. Set the following environment variables before running this script:
#    GORM_ORACLEDB_USER,
#    GORM_ORACLEDB_PASSWORD,
#    GORM_ORACLEDB_CONNECTSTRING,
#    GORM_ORACLEDB_LIBDIR
# 2. Run the script ./run_tests.sh

# Check for required Oracle environment variables
export GORM_DSN='user="system" password="manager" \
connectString="phoenix793630.dev3sub1phx.databasede3phx.oraclevcn.com:1661/cdb1_pdb1.regress.rdbms.dev.us.oracle.com" \
libDir="/home/hkhasnis/projects/pkgs/23IC"'

# Check if passed-tests.txt exists
if [ ! -f "passed-tests.txt" ]; then
    echo "Error: passed-tests.txt not found!"
    exit 1
fi

# Create a temporary file for the updated test list
temp_file=$(mktemp)
updated_file="passed-tests.txt.new"

echo "Processing tests from passed-tests.txt..."
echo ""

# Read the file line by line
while IFS= read -r line; do
    # Skip empty lines - keep them as is
    if [[ -z "$line" ]]; then
        echo "$line" >> "$temp_file"
        continue
    fi
    
    # Check if line starts with #
    if [[ "$line" =~ ^#(.*)$ ]]; then
        # Extract test name (remove #)
        test_name="${BASH_REMATCH[1]}"
        commented=true
    else
        # Line doesn't start with #
        test_name="$line"
        commented=false
    fi
    
    # Skip if test_name is empty after removing #
    if [[ -z "$test_name" ]]; then
        echo "$line" >> "$temp_file"
        continue
    fi
    
    echo "Testing: $test_name"
    
    # Run the specific test
    if go test -run "^${test_name}$" -v > /dev/null 2>&1; then
        # Test passed - remove # if it was commented
        if $commented; then
            echo "  ✓ PASSED - Uncommenting test"
            echo "$test_name" >> "$temp_file"
        else
            echo "  ✓ PASSED - Already uncommented"
            echo "$test_name" >> "$temp_file"
        fi
    else
        # Test failed - add # if it wasn't commented
        if $commented; then
            echo "  ✗ FAILED - Already commented"
            echo "#$test_name" >> "$temp_file"
        else
            echo "  ✗ FAILED - Commenting out test"
            echo "#$test_name" >> "$temp_file"
        fi
    fi
done < "passed-tests.txt"

# Replace the original file with the updated one
mv "$temp_file" "$updated_file"

echo ""
echo "Test processing complete!"
echo "Updated test list saved as: $updated_file"
echo ""

# Show summary of changes
echo "=== SUMMARY ==="
uncommented_count=$(grep -c '^[^#]' "$updated_file" | grep -v '^$' || echo "0")
commented_count=$(grep -c '^#' "$updated_file" || echo "0")

echo "Uncommented tests (passing): $uncommented_count"
echo "Commented tests (failing): $commented_count"

echo ""
echo "To use the updated file, run:"
echo "mv $updated_file passed-tests.txt"
