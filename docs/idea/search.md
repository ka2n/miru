# miru Package Search Feature

## Overview

Add a package search feature to miru, enabling users to search for libraries across multiple languages and registries. This feature will allow users to discover packages based on specific functionalities or use cases, evaluate their quality and reliability, and view their documentation.

## Objectives

- Integrated search across multiple languages and package registries
- Search based on specific use cases, such as "I want to control concurrency with promises in npm"
- Evaluate packages based on important metrics like popularity, freshness, and backing organization
- Prioritize packages from trusted developers (e.g., sindresorhus)

## Search Sources

Leverage existing search API services:

1. **Libraries.io API**

   - Covers numerous languages and package managers
   - Rich metadata including package dependencies, popularity, and maintenance information
   - Example: `https://libraries.io/api/search?q=promise+limit&platforms=npm`

2. **Package Registry APIs**

   - npm Registry API: `https://registry.npmjs.org/-/v1/search?text=promise+limit+concurrency`
   - RubyGems API: `https://rubygems.org/api/v1/search.json?query=promise`
   - Crates.io API: `https://crates.io/api/v1/crates?q=promise`
   - Go Packages (pkg.go.dev): No direct API, combine with GitHub API for search

3. **GitHub API**
   - Developer information, star count, contributor details
   - Example: `https://api.github.com/search/repositories?q=promise+limit+language:javascript`

## Evaluation Criteria

Key metrics used for evaluating and sorting search results:

### 1. Popularity

- Download/installation count
- GitHub star count
- Dependency count (references from other packages)
- Number of projects using the package

### 2. Freshness

- Last update date
- Release frequency
- Latest version release date
- Active issue/PR response status

### 3. Backing Organization

- Maintainer information (individual vs organization)
- Notable developers (e.g., sindresorhus)
- Major companies/organizations (Microsoft, Google, Facebook, etc.)
- Contributor count and diversity

## Command Line Interface

```
miru search [query] [--lang lang] [--sort criteria]
```

Options:

- `--lang`, `-l`: Limit search to a specific language (e.g., `--lang js`)
- `--sort`, `-s`: Sort criteria for results (`popularity`, `freshness`, `relevance`)
- `--limit`, `-n`: Number of results to display

Example:

```
miru search "promise limit concurrency" --lang js --sort popularity
```

Search results display:

```
No. | Package           | Description                                | Language | Popularity | Freshness | Organization
----+-------------------+--------------------------------------------+----------+------------+-----------+-------------
1   | p-limit           | Run multiple promise-returning & async...  | JS       | ★★★★★      | ★★★★☆     | sindresorhus
2   | promise-pool      | Runs multiple promise-returning & async... | JS       | ★★★★☆      | ★★★★★     | supercharge
3   | async             | Higher-order functions and common patte... | JS       | ★★★★★      | ★★★☆☆     | caolan
...
```

## Implementation Approach

1. **Search Service Abstraction**

   - Unified interface for various search sources (Libraries.io, npm, GitHub, etc.)
   - Concurrent searching for fast result retrieval

2. **Result Integration and Deduplication**

   - Merge results from multiple sources
   - Deduplicate based on package name, repository URL, etc.
   - Enrich results by combining information from different sources

3. **Scoring System**

   - Calculate scores based on popularity, freshness, and organizational reliability
   - Bonus scores for packages from notable developers/organizations
   - Ranking according to user-specified sorting criteria

4. **Interactive Selection**

   - Enter a number from search results to select a package
   - Display documentation for the selected package

5. **MCP Server Integration**
   - Add `search_packages` tool
   - Enable search functionality from AI assistants

## Future Expansion Possibilities

1. **Extended Search Filters**

   - Filtering by license type
   - Filtering by dependency count/size
   - Filtering by maintenance status

2. **Semantic Search**

   - Understanding natural language queries
   - Converting "Promise with parallel processing" to "promise concurrency limit"
   - Searching based on package functionality

3. **Package Comparison**

   - Feature comparison of similar packages
   - Display benchmark results
   - Dependency graph visualization

4. **Local Cache and History**

   - Cache search results
   - Save search history
   - Bookmark frequently used packages

5. **Community Feedback Integration**
   - Mention count on Stack Overflow
   - References in blog posts and tutorials
   - Security vulnerability information
