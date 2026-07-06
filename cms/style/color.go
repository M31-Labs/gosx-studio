package style

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	minBodyContrast  = 4.5
	minMutedContrast = 3.0
)

type rgb struct {
	r float64
	g float64
	b float64
}

func validateTokenContrast(tokens TokenSet) ValidationErrors {
	var errs ValidationErrors
	canvas, ok := parseHexColor(tokens.Colors["canvas"])
	if !ok {
		return errs
	}
	checks := []struct {
		key   string
		min   float64
		label string
	}{
		{key: "ink", min: minBodyContrast, label: "primary text"},
		{key: "ink-soft", min: minBodyContrast, label: "secondary text"},
		{key: "muted", min: minMutedContrast, label: "muted text"},
	}
	for _, check := range checks {
		foreground, ok := parseHexColor(tokens.Colors[check.key])
		if !ok {
			continue
		}
		ratio := contrastRatio(canvas, foreground)
		if ratio < check.min {
			errs = append(errs, ValidationError{
				Path:    "tokens.colors." + check.key,
				Message: fmt.Sprintf("%s contrast %.2f is below %.1f against canvas", check.label, ratio, check.min),
			})
		}
	}
	return errs
}

func normalizeColor(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	if !strings.HasPrefix(value, "#") && (len(value) == 3 || len(value) == 6) {
		value = "#" + value
	}
	color, ok := parseHexColor(value)
	if !ok {
		return ""
	}
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(color.r*255)), int(math.Round(color.g*255)), int(math.Round(color.b*255)))
}

func parseHexColor(value string) (rgb, bool) {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "#")
	if len(value) == 3 {
		value = string([]byte{value[0], value[0], value[1], value[1], value[2], value[2]})
	}
	if len(value) != 6 {
		return rgb{}, false
	}
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return rgb{}, false
	}
	return rgb{
		r: float64((parsed>>16)&0xff) / 255,
		g: float64((parsed>>8)&0xff) / 255,
		b: float64(parsed&0xff) / 255,
	}, true
}

func contrastRatio(a, b rgb) float64 {
	la := relativeLuminance(a)
	lb := relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(color rgb) float64 {
	return 0.2126*linearChannel(color.r) + 0.7152*linearChannel(color.g) + 0.0722*linearChannel(color.b)
}

func linearChannel(value float64) float64 {
	if value <= 0.03928 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}
