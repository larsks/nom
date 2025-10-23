package commands

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/sahilm/fuzzy"

	"github.com/guyfedwards/nom/v2/internal/config"
)

type FilterTerm struct {
	Title     string
	FeedNames []string
	Tags      []string
}

// Struct to aid in filtering items into ranks for BubbleTea
type Filterer struct {
	FeedNames []string
	Tags      []string
	Term      FilterTerm
	Config    config.Config
}

// Breaks what's returned from TUIItem.FilterValue() into a TUIItem.
func (f *Filterer) GetItem(filterValue string) TUIItem {
	splits := strings.Split(filterValue, "||")

	return TUIItem{
		Title:    splits[0],
		FeedName: strings.ToLower(splits[1]),
		Tags:     splits[2:],
	}
}

// Extracts `tag:.*` from the stored f.Term.Title
func (f *Filterer) ExtractFiltersFor(tags ...string) []string {
	var extractedTags []string
	done := false
	for !done {
		// `complete` matches 3 potential capture groups after tags, in which
		// `[^"]` matches a character that isn't a `"`, `[^']` that isn't a `'`,
		// etc. If it's no quotes, you can also do `feed:with\ spaces`
		// `incomplete` matches unfinished quoted tags and removes them from the
		// search. The order of the capture groups MATTERS.
		// In both examples, the %s section matches all potential tag aliases
		// passed in for one tag.
		complete := regexp.MustCompile(fmt.Sprintf(`(%s):("([^"]+)"|'([^']+)'|(([^\\ ]|\\ )+))`, strings.Join(tags, "|")))
		incomplete := regexp.MustCompile(fmt.Sprintf(`(%s):("[^"]*|'[^']*)`, strings.Join(tags, "|")))

		matches := complete.FindStringSubmatch(f.Term.Title)

		match := ""
		if matches != nil {
			// double quotes
			if matches[3] != "" {
				match = matches[3]
				// single quotes
			} else if matches[4] != "" {
				match = matches[4]
				// no quotes
			} else if matches[5] != "" {
				match = strings.ReplaceAll(matches[5], `\ `, " ")
			}
			f.Term.Title = strings.Replace(f.Term.Title, matches[0], "", 1)
		} else {
			// fallback to regular matching without filter
			matches = incomplete.FindStringSubmatch(f.Term.Title)
			if matches != nil {
				f.Term.Title = strings.Replace(f.Term.Title, matches[0], "", 1)
			}
			done = true
		}

		if match != "" {
			extractedTags = append(extractedTags, strings.ToLower(match))
		}
	}
	if f.Term.Title == "" {
		f.Term.Title = " "
	}

	return extractedTags
}

// filterByConstraints filters candidates by requiring ALL filter values to match
// Returns a map of indices that match all constraints with their best scores
func (f *Filterer) filterByConstraints(filterValues []string, targets []string, extractField func(TUIItem) string) map[int]int {
	if len(filterValues) == 0 {
		return nil
	}

	// For each target, check if it matches ALL filter values (AND logic)
	matchedIndices := make(map[int]int)

	for idx, target := range targets {
		item := f.GetItem(target)
		fieldValue := extractField(item)

		// Check if this item matches ALL filter values
		allMatch := true
		totalScore := 0

		for _, filterVal := range filterValues {
			matches := fuzzy.Find(filterVal, []string{fieldValue})
			if len(matches) == 0 || matches[0].Score == 0 {
				allMatch = false
				break
			}
			totalScore += matches[0].Score
		}

		if allMatch {
			matchedIndices[idx] = totalScore
		}
	}

	return matchedIndices
}

// Runs all filters with combined query support
func (f *Filterer) Filter(targets []string) []fuzzy.Match {
	// Build target fields for all items
	var targetTitles []string
	for _, target := range targets {
		i := f.GetItem(target)
		title := i.Title
		if f.Config.Filtering.DefaultIncludeFeedName {
			title = strings.Join([]string{i.FeedName, i.Title}, " ")
		}
		targetTitles = append(targetTitles, title)
	}

	// Start with all indices as candidates
	candidateIndices := make(map[int]int) // index -> score
	for idx := range targets {
		candidateIndices[idx] = 0
	}

	// Apply feed name filter as hard constraint (AND logic for multiple feeds)
	if len(f.FeedNames) > 0 {
		feedMatches := f.filterByConstraints(f.FeedNames, targets, func(item TUIItem) string {
			return item.FeedName
		})
		if len(feedMatches) == 0 {
			// No matches for feed filter, return empty results
			return []fuzzy.Match{}
		}
		// Update candidates to intersection
		candidateIndices = feedMatches
	}

	// Apply tag filter as hard constraint (AND logic for multiple tags)
	if len(f.Tags) > 0 {
		tagMatches := f.filterByConstraints(f.Tags, targets, func(item TUIItem) string {
			return strings.Join(item.Tags, " ")
		})
		if len(tagMatches) == 0 {
			// No matches for tag filter, return empty results
			return []fuzzy.Match{}
		}

		// Intersect with existing candidates
		if len(f.FeedNames) > 0 {
			// We already have feed constraints, intersect them
			newCandidates := make(map[int]int)
			for idx := range candidateIndices {
				if score, exists := tagMatches[idx]; exists {
					newCandidates[idx] = candidateIndices[idx] + score
				}
			}
			candidateIndices = newCandidates
		} else {
			// Only tag constraint
			candidateIndices = tagMatches
		}
	}

	// If no results after constraint filtering, return empty
	if len(candidateIndices) == 0 {
		return []fuzzy.Match{}
	}

	// Build list of candidate targets for title search
	var candidateTargets []string
	var candidateMapping []int // maps result index to original index
	for idx := range targets {
		if _, exists := candidateIndices[idx]; exists {
			candidateTargets = append(candidateTargets, targetTitles[idx])
			candidateMapping = append(candidateMapping, idx)
		}
	}

	// Apply title fuzzy search on remaining candidates
	var ranks fuzzy.Matches
	trimmedTitle := strings.TrimSpace(f.Term.Title)
	if trimmedTitle != "" {
		// Perform fuzzy search on title
		titleMatches := fuzzy.Find(trimmedTitle, candidateTargets)

		// Combine title scores with constraint scores
		for _, match := range titleMatches {
			originalIdx := candidateMapping[match.Index]
			constraintScore := candidateIndices[originalIdx]

			ranks = append(ranks, fuzzy.Match{
				Str:            match.Str,
				Index:          originalIdx,
				MatchedIndexes: match.MatchedIndexes,
				Score:          match.Score + constraintScore,
			})
		}
	} else {
		// No title search, return all candidates ranked by constraint scores
		// Iterate in order to maintain stable results
		for _, idx := range candidateMapping {
			score := candidateIndices[idx]
			ranks = append(ranks, fuzzy.Match{
				Str:            targetTitles[idx],
				Index:          idx,
				MatchedIndexes: []int{},
				Score:          score,
			})
		}
	}

	// Sort by score (descending), then by original index (ascending) for stability
	slices.SortStableFunc(ranks, func(left fuzzy.Match, right fuzzy.Match) int {
		// First compare by score (descending)
		if left.Score != right.Score {
			return right.Score - left.Score
		}
		// For equal scores, maintain original order by index (ascending)
		return left.Index - right.Index
	})

	return ranks
}

func NewFilterer(term string, config config.Config) Filterer {
	f := Filterer{
		Config: config,
		Term: FilterTerm{
			Title: term,
		},
	}

	f.FeedNames = f.ExtractFiltersFor("feedname", "feed", "f")
	f.Tags = f.ExtractFiltersFor("tag", "t")

	return f
}

func CustomFilter(config config.Config) list.FilterFunc {
	return func(term string, targets []string) []list.Rank {
		filterer := NewFilterer(term, config)

		ranks := filterer.Filter(targets)

		result := make([]list.Rank, len(ranks))
		for i, rank := range ranks {
			result[i] = list.Rank{
				Index:          rank.Index,
				MatchedIndexes: rank.MatchedIndexes,
			}
		}

		return result
	}
}
