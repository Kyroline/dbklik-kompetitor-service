package services

import (
	"math"
	"sort"
	"strings"

	"github.com/dbklik/dbklik-kompetitor-service/modules/kompetitor/domain/repositories"
)

// Match is one item ↔ kompetitor-product match above the threshold.
type Match struct {
	Product repositories.MatchableProduct
	Score   float64
}

// TextMatcher matches our products against scraped kompetitor products —
// a port of App\Services\ProductMappingService's scoring core, itself a
// port of the grouping_dbklik_hpp notebook:
//
//  1. Candidate kompetitors per item are limited by the (kategori, brand)
//     mapping (supplied as allowedByItem — this type never touches the DB).
//  2. Product names are cleaned (uppercase, alphanumeric only) and turned
//     into TF-IDF vectors; similarity is cosine similarity.
//  3. Only the best match per (item × kompetitor) above the threshold is kept.
type TextMatcher struct{}

func NewTextMatcher() *TextMatcher { return &TextMatcher{} }

// Tokenize is the equivalent of clean_text in the notebook: uppercase,
// alphanumeric only, collapsed whitespace.
func (TextMatcher) Tokenize(name string) []string {
	upper := strings.ToUpper(name)

	var b strings.Builder
	b.Grow(len(upper))
	for _, r := range upper {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte(' ')
	}

	return strings.Fields(b.String())
}

// ComputeIDF derives token IDF weights from the combined corpora (scraped
// product names + our item names). Formula identical to the PHP port:
// log((1 + N) / (1 + df)) + 1.
func (m TextMatcher) ComputeIDF(corpora ...[]string) map[string]float64 {
	df := map[string]int{}
	totalDocs := 0

	for _, names := range corpora {
		for _, name := range names {
			totalDocs++
			seen := map[string]bool{}
			for _, token := range m.Tokenize(name) {
				if seen[token] {
					continue
				}
				seen[token] = true
				df[token]++
			}
		}
	}

	idf := make(map[string]float64, len(df))
	for token, count := range df {
		idf[token] = math.Log(float64(1+totalDocs)/float64(1+count)) + 1
	}

	return idf
}

// Vectorize builds an L2-normalized TF-IDF vector (token => weight).
// Tokens absent from the corpus weigh 0, so a fallback batch must be part
// of the IDF corpus for its products to score at all.
func (TextMatcher) vectorize(tokens []string, idf map[string]float64) map[string]float64 {
	if len(tokens) == 0 {
		return nil
	}

	tf := map[string]int{}
	for _, token := range tokens {
		tf[token]++
	}

	vector := make(map[string]float64, len(tf))
	norm := 0.0
	for token, count := range tf {
		w := float64(count) * idf[token]
		vector[token] = w
		norm += w * w
	}

	norm = math.Sqrt(norm)
	if norm == 0 {
		return nil
	}
	for token, w := range vector {
		vector[token] = w / norm
	}

	return vector
}

// ComputeMatches is the TF-IDF cosine-similarity core: the best match per
// (item × kompetitor) above the threshold, restricted to the kompetitors
// mapped to the item's (kategori, brand) pair.
//
// allowedByItem is [item_id => [kompetitor_id => true]]; items missing from
// it are skipped entirely. Keeping it a parameter leaves this function pure
// and easy to test.
func (m TextMatcher) ComputeMatches(
	items []repositories.MappableItem,
	products []repositories.MatchableProduct,
	threshold float64,
	idf map[string]float64,
	allowedByItem map[uint64]map[uint64]bool,
) map[uint64]map[uint64]Match {
	if len(items) == 0 || len(products) == 0 {
		return map[uint64]map[uint64]Match{}
	}

	if idf == nil {
		productNames := make([]string, len(products))
		for i, p := range products {
			productNames[i] = p.Name
		}
		itemNames := make([]string, len(items))
		for i, it := range items {
			itemNames[i] = it.NamaAccurate
		}
		idf = m.ComputeIDF(productNames, itemNames)
	}

	// Inverted index over the kompetitor products, so scoring only touches
	// products that share a token with the item.
	invertedIndex := map[string]map[int]float64{}
	for i, product := range products {
		for token, w := range m.vectorize(m.Tokenize(product.Name), idf) {
			if invertedIndex[token] == nil {
				invertedIndex[token] = map[int]float64{}
			}
			invertedIndex[token][i] = w
		}
	}

	result := make(map[uint64]map[uint64]Match)

	for _, item := range items {
		allowed := allowedByItem[item.ID]
		if len(allowed) == 0 {
			continue
		}

		vector := m.vectorize(m.Tokenize(item.NamaAccurate), idf)
		if len(vector) == 0 {
			continue
		}

		// Accumulate the dot product only over products sharing a token.
		scores := map[int]float64{}
		var touched []int
		for token, w := range vector {
			for productIdx, pw := range invertedIndex[token] {
				if _, seen := scores[productIdx]; !seen {
					touched = append(touched, productIdx)
				}
				scores[productIdx] += w * pw
			}
		}

		// Product order decides ties, so walk the indices in order rather
		// than in Go's randomized map order.
		sort.Ints(touched)

		best := map[uint64]Match{}
		for _, productIdx := range touched {
			score := scores[productIdx]
			if score <= threshold {
				continue
			}

			product := products[productIdx]
			if !allowed[product.KompetitorID] {
				continue
			}

			if existing, ok := best[product.KompetitorID]; ok && score <= existing.Score {
				continue
			}
			best[product.KompetitorID] = Match{Product: product, Score: math.Min(score, 1)}
		}

		if len(best) > 0 {
			result[item.ID] = best
		}
	}

	return result
}
