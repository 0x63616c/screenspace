package main

// SeedVideo describes a wallpaper to seed into the dev environment.
type SeedVideo struct {
	URL        string   // Direct CDN download URL (HD 1080p)
	Title      string
	Category   string
	Tags       []string
	Downloads  int64    // fake download count for Popular ordering
}

// videos is the manifest of wallpapers to seed. Uses Pexels CDN URLs
// for HD (1920x1080) versions which are typically 5-15MB each.
// The actual resolution, duration, and file size are probed from the
// downloaded file via ffprobe.
var videos = []SeedVideo{
	// Nature (3)
	{
		URL:       "https://videos.pexels.com/video-files/857251/857251-hd_1620_1080_25fps.mp4",
		Title:     "Ocean Waves at Sunset",
		Category:  "nature",
		Tags:      []string{"ocean", "waves", "sunset", "water"},
		Downloads: 342,
	},
	{
		URL:       "https://videos.pexels.com/video-files/2491284/2491284-hd_2048_1080_24fps.mp4",
		Title:     "Misty Mountain Forest",
		Category:  "nature",
		Tags:      []string{"forest", "mountains", "mist", "trees"},
		Downloads: 287,
	},
	{
		URL:       "https://videos.pexels.com/video-files/1448735/1448735-hd_2048_1080_24fps.mp4",
		Title:     "Autumn Leaves Falling",
		Category:  "nature",
		Tags:      []string{"autumn", "leaves", "fall", "seasonal"},
		Downloads: 198,
	},

	// Abstract (2)
	{
		URL:       "https://videos.pexels.com/video-files/3141210/3141210-hd_1920_1080_25fps.mp4",
		Title:     "Flowing Ink in Water",
		Category:  "abstract",
		Tags:      []string{"ink", "water", "fluid", "colorful"},
		Downloads: 456,
	},
	{
		URL:       "https://videos.pexels.com/video-files/3129671/3129671-hd_1920_1080_30fps.mp4",
		Title:     "Abstract Light Geometry",
		Category:  "abstract",
		Tags:      []string{"geometry", "lights", "digital", "lines"},
		Downloads: 523,
	},

	// Space (2)
	{
		URL:       "https://videos.pexels.com/video-files/1851190/1851190-hd_1920_1080_25fps.mp4",
		Title:     "Milky Way Timelapse",
		Category:  "space",
		Tags:      []string{"milky way", "stars", "night sky", "timelapse"},
		Downloads: 612,
	},
	{
		URL:       "https://videos.pexels.com/video-files/856356/856356-hd_1920_1080_25fps.mp4",
		Title:     "Earth From Space",
		Category:  "space",
		Tags:      []string{"earth", "space", "rotation", "planet"},
		Downloads: 389,
	},

	// Urban (2)
	{
		URL:       "https://videos.pexels.com/video-files/1721294/1721294-hd_1920_1080_25fps.mp4",
		Title:     "City Skyline at Dusk",
		Category:  "urban",
		Tags:      []string{"city", "skyline", "dusk", "buildings"},
		Downloads: 275,
	},
	{
		URL:       "https://videos.pexels.com/video-files/1826896/1826896-hd_1920_1080_24fps.mp4",
		Title:     "Aerial City View",
		Category:  "urban",
		Tags:      []string{"aerial", "city", "drone", "urban"},
		Downloads: 441,
	},

	// Underwater (1)
	{
		URL:       "https://videos.pexels.com/video-files/855029/855029-hd_1920_1080_30fps.mp4",
		Title:     "Coral Reef Fish",
		Category:  "underwater",
		Tags:      []string{"coral", "reef", "fish", "underwater", "ocean"},
		Downloads: 167,
	},
}
