package config

import (
	"fmt"
	"strconv"
	"strings"
)

// Named colors → R,G,B
var namedColors = map[string][3]int{
	"red":    {239, 68, 68},
	"orange": {249, 115, 22},
	"yellow": {234, 179, 8},
	"green":  {34, 197, 94},
	"teal":   {0, 168, 120},
	"blue":   {59, 130, 246},
	"indigo": {99, 102, 241},
	"purple": {139, 92, 246},
	"pink":   {236, 72, 153},
	"gray":   {107, 114, 128},
	"white":  {255, 255, 255},
}

// ResolveColor takes a color name or "R,G,B" string and returns "R,G,B".
// Returns the input unchanged if it's already in R,G,B format.
func ResolveColor(input string) (string, error) {
	lower := strings.ToLower(strings.TrimSpace(input))

	// Check named colors
	if rgb, ok := namedColors[lower]; ok {
		return fmt.Sprintf("%d,%d,%d", rgb[0], rgb[1], rgb[2]), nil
	}

	// Validate R,G,B format
	parts := strings.SplitN(input, ",", 3)
	if len(parts) == 3 {
		for _, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil || n < 0 || n > 255 {
				return "", fmt.Errorf("invalid color value: %s (must be 0-255)", p)
			}
		}
		return input, nil
	}

	names := make([]string, 0, len(namedColors))
	for k := range namedColors {
		names = append(names, k)
	}
	return "", fmt.Errorf("unknown color '%s' (use a name like %s, or R,G,B)", input, strings.Join(names, ", "))
}

// ColorNames returns all available named colors.
func ColorNames() []string {
	return []string{"red", "orange", "yellow", "green", "teal", "blue", "indigo", "purple", "pink", "gray", "white"}
}
