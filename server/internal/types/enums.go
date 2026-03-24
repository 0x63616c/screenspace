package types

// WallpaperStatus represents the lifecycle state of a wallpaper.
type WallpaperStatus string

// WallpaperStatus values.
const (
	StatusPending       WallpaperStatus = "pending"
	StatusPendingReview WallpaperStatus = "pending_review"
	StatusApproved      WallpaperStatus = "approved"
	StatusRejected      WallpaperStatus = "rejected"
)

// Valid returns true if the status is a known value.
func (s WallpaperStatus) Valid() bool {
	switch s {
	case StatusPending, StatusPendingReview, StatusApproved, StatusRejected:
		return true
	}
	return false
}

// UserRole represents the access level of a user account.
type UserRole string

// UserRole values.
const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// Valid returns true if the role is a known value.
func (r UserRole) Valid() bool {
	switch r {
	case RoleUser, RoleAdmin:
		return true
	}
	return false
}

// Category represents the content category of a wallpaper.
type Category string

// Category values.
const (
	CategoryNature     Category = "nature"
	CategoryAbstract   Category = "abstract"
	CategoryUrban      Category = "urban"
	CategoryCinematic  Category = "cinematic"
	CategorySpace      Category = "space"
	CategoryUnderwater Category = "underwater"
	CategoryMinimal    Category = "minimal"
	CategoryOther      Category = "other"
)

// AllCategories returns every valid Category value.
func AllCategories() []Category {
	return []Category{
		CategoryNature,
		CategoryAbstract,
		CategoryUrban,
		CategoryCinematic,
		CategorySpace,
		CategoryUnderwater,
		CategoryMinimal,
		CategoryOther,
	}
}

// Valid returns true if the category is a known value.
func (c Category) Valid() bool {
	switch c {
	case CategoryNature, CategoryAbstract, CategoryUrban, CategoryCinematic,
		CategorySpace, CategoryUnderwater, CategoryMinimal, CategoryOther:
		return true
	}
	return false
}

// SortOrder controls list ordering for wallpaper queries.
type SortOrder string

// SortOrder values.
const (
	SortRecent  SortOrder = "recent"
	SortPopular SortOrder = "popular"
)

// Valid returns true if the sort order is a known value.
func (s SortOrder) Valid() bool {
	switch s {
	case SortRecent, SortPopular:
		return true
	}
	return false
}
