#!/bin/bash

# Add missing imports to vector_db.go
echo "Updating vector_db.go imports..."
sed -i '10i\	"math"\n\	"sort"' /home/bleepbloop/deltacli/vector_db.go

# Add missing imports to embedding_manager.go
echo "Updating embedding_manager.go imports..."
sed -i '10i\	"math"\n\	"sort"' /home/bleepbloop/deltacli/embedding_manager.go

# Add missing imports to speculative_decoding.go
echo "Updating speculative_decoding.go imports..."
sed -i '10i\	"math"' /home/bleepbloop/deltacli/speculative_decoding.go

echo "Updating vector_db.go SQLite query..."
sed -i 's/\.replace("\${dimension}", fmt\.Sprintf("%d", vm\.config\.EmbeddingDimension))//' /home/bleepbloop/deltacli/vector_db.go

echo "Done!"