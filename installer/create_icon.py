#!/usr/bin/env python3
"""
Create Delta icon for Windows installer
Creates a simple icon with the Delta symbol
"""

import os
import sys

try:
    from PIL import Image, ImageDraw, ImageFont
except ImportError:
    print("PIL (Pillow) is required. Install with: pip install Pillow")
    sys.exit(1)

def create_delta_icon():
    # Create icon sizes required for Windows .ico
    sizes = [16, 32, 48, 64, 128, 256]
    images = []
    
    for size in sizes:
        # Create a new image with transparent background
        img = Image.new('RGBA', (size, size), (0, 0, 0, 0))
        draw = ImageDraw.Draw(img)
        
        # Calculate dimensions
        margin = size // 8
        triangle_size = size - (2 * margin)
        
        # Define triangle points (equilateral triangle)
        x_center = size // 2
        y_top = margin
        y_bottom = margin + triangle_size
        x_left = x_center - (triangle_size // 2)
        x_right = x_center + (triangle_size // 2)
        
        points = [
            (x_center, y_top),      # Top
            (x_left, y_bottom),     # Bottom left
            (x_right, y_bottom),    # Bottom right
        ]
        
        # Draw filled triangle (Delta symbol)
        # Dark blue color
        fill_color = (0, 51, 102, 255)
        draw.polygon(points, fill=fill_color)
        
        # Draw border
        border_color = (0, 102, 204, 255)
        border_width = max(1, size // 32)
        for i in range(len(points)):
            start = points[i]
            end = points[(i + 1) % len(points)]
            draw.line([start, end], fill=border_color, width=border_width)
        
        images.append(img)
    
    # Save as ICO file
    images[0].save('delta.ico', format='ICO', sizes=[(s, s) for s in sizes])
    print("Created delta.ico successfully")

if __name__ == "__main__":
    create_delta_icon()