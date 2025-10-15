package constants

type Ordering string

// Store constants
const (
	AscendingOrdering  Ordering = "asc"
	DescendingOrdering Ordering = "desc"
	DefaultOrdering    Ordering = AscendingOrdering
)
