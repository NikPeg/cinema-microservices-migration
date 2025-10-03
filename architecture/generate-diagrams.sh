#!/bin/bash

# Script to generate diagram images from PlantUML files
# Requires PlantUML to be installed

echo "Generating architecture diagrams..."

# Check if PlantUML is available
if ! command -v plantuml &> /dev/null; then
    echo "PlantUML is not installed. Installing via brew..."
    if command -v brew &> /dev/null; then
        brew install plantuml
    else
        echo "Please install PlantUML manually:"
        echo "  - macOS: brew install plantuml"
        echo "  - Ubuntu/Debian: apt-get install plantuml"
        echo "  - Or download from: http://plantuml.com/download"
        exit 1
    fi
fi

# Generate diagrams
echo "Generating C4 Container Diagram (full)..."
plantuml -tsvg c4-container-diagram.puml
plantuml -tpng c4-container-diagram.puml

echo "Generating C4 Container Diagram (simplified)..."
plantuml -tsvg c4-container-simplified.puml
plantuml -tpng c4-container-simplified.puml

echo "Diagrams generated successfully!"
echo "Available formats:"
echo "  - SVG: *.svg (vector, scalable)"
echo "  - PNG: *.png (raster, fixed resolution)"
