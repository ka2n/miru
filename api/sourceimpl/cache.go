package sourceimpl

import (
	"fmt"

	"github.com/ka2n/miru/api/cache"
	"github.com/ka2n/miru/api/investigator"
	"github.com/ka2n/miru/api/source"
)

// FetchWithCache fetches data from the source with cache support
// It uses the cache.GetOrSet function to retrieve data from cache or fetch it if not available
// The cache key is generated from the investigator type and package path
// The forceUpdate parameter can be used to ignore the cache and fetch fresh data
func FetchWithCache(investigator investigator.SourceInvestigator, packagePath string, forceUpdate bool) (source.Data, error) {
	// Generate cache key
	cacheKey := fmt.Sprintf("%s:%s", investigator.GetSourceType(), packagePath)

	// Create cache instance for source.Data type
	cache := cache.New[source.Data]("fetch")

	// Get data from cache or fetch it
	data, err := cache.GetOrSet(cacheKey, func() (source.Data, error) {
		return investigator.Fetch(packagePath)
	}, forceUpdate)

	return data, err
}
