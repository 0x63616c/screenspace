package main

// SeedVideo describes a wallpaper to seed into the dev environment.
type SeedVideo struct {
	PexelsID   int
	Title      string
	Category   string
	Tags       []string
	Resolution string
	Width      int
	Height     int
	Duration   float64
	Downloads  int64 // fake download count for Popular ordering
}

// videos is the manifest of wallpapers to seed. When PEXELS_API_KEY is set,
// the seed script fetches real videos from the Pexels API using PexelsID.
// Otherwise it generates placeholder videos with ffmpeg.
var videos = []SeedVideo{
	// Nature (3)
	{
		PexelsID:   857251,
		Title:      "Ocean Waves at Sunset",
		Category:   "nature",
		Tags:       []string{"ocean", "waves", "sunset", "water"},
		Resolution: "3840x2160",
		Width:      3840,
		Height:     2160,
		Duration:   10.0,
		Downloads:  342,
	},
	{
		PexelsID:   2491284,
		Title:      "Misty Mountain Forest",
		Category:   "nature",
		Tags:       []string{"forest", "mountains", "mist", "trees"},
		Resolution: "3840x2160",
		Width:      3840,
		Height:     2160,
		Duration:   12.0,
		Downloads:  287,
	},
	{
		PexelsID:   1448735,
		Title:      "Autumn Leaves Falling",
		Category:   "nature",
		Tags:       []string{"autumn", "leaves", "fall", "seasonal"},
		Resolution: "1920x1080",
		Width:      1920,
		Height:     1080,
		Duration:   8.0,
		Downloads:  198,
	},

	// Abstract (2)
	{
		PexelsID:   3141210,
		Title:      "Flowing Ink in Water",
		Category:   "abstract",
		Tags:       []string{"ink", "water", "fluid", "colorful"},
		Resolution: "3840x2160",
		Width:      3840,
		Height:     2160,
		Duration:   10.0,
		Downloads:  456,
	},
	{
		PexelsID:   2795167,
		Title:      "Neon Light Trails",
		Category:   "abstract",
		Tags:       []string{"neon", "lights", "trails", "glow"},
		Resolution: "1920x1080",
		Width:      1920,
		Height:     1080,
		Duration:   8.0,
		Downloads:  523,
	},

	// Space (2)
	{
		PexelsID:   1851190,
		Title:      "Milky Way Timelapse",
		Category:   "space",
		Tags:       []string{"milky way", "stars", "night sky", "timelapse"},
		Resolution: "3840x2160",
		Width:      3840,
		Height:     2160,
		Duration:   15.0,
		Downloads:  612,
	},
	{
		PexelsID:   1722591,
		Title:      "Northern Lights Aurora",
		Category:   "space",
		Tags:       []string{"aurora", "northern lights", "sky", "polar"},
		Resolution: "1920x1080",
		Width:      1920,
		Height:     1080,
		Duration:   12.0,
		Downloads:  389,
	},

	// Urban (2)
	{
		PexelsID:   1721294,
		Title:      "City Skyline at Night",
		Category:   "urban",
		Tags:       []string{"city", "skyline", "night", "buildings"},
		Resolution: "3840x2160",
		Width:      3840,
		Height:     2160,
		Duration:   10.0,
		Downloads:  275,
	},
	{
		PexelsID:   3048163,
		Title:      "Rain on City Streets",
		Category:   "urban",
		Tags:       []string{"rain", "city", "streets", "reflections"},
		Resolution: "1920x1080",
		Width:      1920,
		Height:     1080,
		Duration:   8.0,
		Downloads:  441,
	},

	// Underwater (1)
	{
		PexelsID:   855029,
		Title:      "Coral Reef Fish",
		Category:   "underwater",
		Tags:       []string{"coral", "reef", "fish", "underwater", "ocean"},
		Resolution: "1920x1080",
		Width:      1920,
		Height:     1080,
		Duration:   10.0,
		Downloads:  167,
	},
}
