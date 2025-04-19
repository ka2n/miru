package api

import (
	"fmt"
	"time"

	"github.com/ka2n/miru/api/source"
	"github.com/ka2n/miru/api/sourceimpl"
	"github.com/ka2n/miru/api/sourceresolver"
)

// Investigation is a structure that represents data under investigation
type Investigation struct {
	// Query is the initial query
	Query InitialQuery

	// CollectedData is data collected from each source
	CollectedData map[source.Type]source.Data
}

// NewInvestigation creates a new investigation
func NewInvestigation(query InitialQuery) *Investigation {
	return &Investigation{
		Query:         query,
		CollectedData: make(map[source.Type]source.Data),
	}
}

func (i *Investigation) Do() error {
	queue := []source.Reference{
		i.Query.SourceRef,
	}

	for len(queue) > 0 {
		sourceRef := queue[0]
		queue = queue[1:]
		investigator := sourceresolver.Investigator(sourceRef.Type)
		if investigator == nil {
			return fmt.Errorf("investigator not found for source type: %s", sourceRef.Type)
		}
		data, err := sourceimpl.FetchWithCache(investigator, sourceRef.Path, i.Query.ForceUpdate)
		if err != nil {
			i.CollectedData[sourceRef.Type] = source.Data{
				FetchError: err,
				FetchedAt:  time.Now(),
			}
			continue
		}
		data.Source = sourceRef

		i.CollectedData[sourceRef.Type] = data

		// Enough data collected, stop investigation
		if i.IsSufficient() {
			return nil
		}

		// Add related sources to the queue
		for _, r := range data.RelatedSources {
			ref := r.ToSourceReference()
			if _, ok := i.CollectedData[ref.Type]; !ok {
				queue = append(queue, ref)
			}
		}
	}

	return nil
}

func (i Investigation) IsSufficient() bool {
	return false
}
