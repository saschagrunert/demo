package demo

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// Theme selects the color palette of a run.
type Theme string

const (
	// ThemeAuto picks ThemeLight if the COLORFGBG environment variable reports
	// a light terminal background, and ThemeDark otherwise. It is the default.
	ThemeAuto Theme = "auto"

	// ThemeDark is the theme for terminal backgrounds using dark colors.
	ThemeDark Theme = "dark"

	// ThemeLight is the theme for terminal backgrounds using light colors.
	ThemeLight Theme = "light"
)

const (
	lightBackgroundThreshold = 9
	lightBackgroundMaximum   = 15
	lightBackgroundIndex     = 7
)

var errInvalidTheme = errors.New("invalid theme")

// Palette defines the terminal color attributes used by a run. The attributes
// are the ones of the github.com/fatih/color package. Empty fields fall back
// to the default palette of the selected theme.
type Palette struct {
	Title       []color.Attribute
	Description []color.Attribute
	Command     []color.Attribute
}

// DefaultDarkPalette returns the default palette for dark backgrounds.
func DefaultDarkPalette() Palette {
	return Palette{
		Title:       []color.Attribute{color.FgCyan},
		Description: []color.Attribute{color.FgWhite, color.Faint},
		Command:     []color.Attribute{color.FgGreen},
	}
}

// DefaultLightPalette returns the default high-contrast palette for light backgrounds.
func DefaultLightPalette() Palette {
	return Palette{
		Title:       []color.Attribute{color.FgBlue},
		Description: []color.Attribute{color.FgHiBlue},
		Command:     []color.Attribute{color.FgGreen},
	}
}

// colors holds the formatting functions for the parts of a run.
type colors struct {
	title       func(format string, a ...any) string
	description func(format string, a ...any) string
	command     func(format string, a ...any) string
}

// newColors returns the formatting functions for the provided options, whose
// theme has to be resolved already.
func newColors(opts *Options) colors {
	if opts.NoColor {
		return colors{
			title:       fmt.Sprintf,
			description: fmt.Sprintf,
			command:     fmt.Sprintf,
		}
	}

	palette := mergePaletteWithDefaults(opts.Theme, opts.Palette)

	return colors{
		title:       color.New(palette.Title...).SprintfFunc(),
		description: color.New(palette.Description...).SprintfFunc(),
		command:     color.New(palette.Command...).SprintfFunc(),
	}
}

func mergePaletteWithDefaults(theme Theme, palette Palette) Palette {
	defaults := DefaultDarkPalette()
	if theme == ThemeLight {
		defaults = DefaultLightPalette()
	}

	if len(palette.Title) == 0 {
		palette.Title = defaults.Title
	}

	if len(palette.Description) == 0 {
		palette.Description = defaults.Description
	}

	if len(palette.Command) == 0 {
		palette.Command = defaults.Command
	}

	return palette
}

// resolveTheme validates the theme and turns ThemeAuto, or an empty theme,
// into ThemeDark or ThemeLight.
func resolveTheme(theme Theme) (Theme, error) {
	normalized, err := normalizeTheme(theme)
	if err != nil {
		return "", err
	}

	if normalized != ThemeAuto {
		return normalized, nil
	}

	if detectLightBackground() {
		return ThemeLight, nil
	}

	return ThemeDark, nil
}

func normalizeTheme(theme Theme) (Theme, error) {
	switch normalized := Theme(strings.ToLower(strings.TrimSpace(string(theme)))); normalized {
	case "", ThemeAuto:
		return ThemeAuto, nil
	case ThemeDark, ThemeLight:
		return normalized, nil
	default:
		return "", fmt.Errorf("%w %q: allowed values are auto, dark, light", errInvalidTheme, theme)
	}
}

func validateTheme(theme string) error {
	_, err := normalizeTheme(Theme(theme))

	return err
}

func detectLightBackground() bool {
	colorFGBG := os.Getenv("COLORFGBG")
	if colorFGBG == "" {
		return false
	}

	parts := strings.Split(colorFGBG, ";")

	bg, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return false
	}

	return bg == lightBackgroundIndex ||
		(bg >= lightBackgroundThreshold && bg <= lightBackgroundMaximum)
}
