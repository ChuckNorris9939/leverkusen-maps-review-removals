package main

import "nuernberg-maps-review-removals/internal/mapsreview"

func discoverySeen(seen map[string]bool, discovery mapsreview.Discovery) bool {
	for _, alias := range mapsreview.PlaceAliases(discovery.ID, discovery.URL) {
		if seen[alias] {
			return true
		}
	}
	return false
}

func markDiscoverySeen(seen map[string]bool, discovery mapsreview.Discovery) {
	for _, alias := range mapsreview.PlaceAliases(discovery.ID, discovery.URL) {
		seen[alias] = true
	}
}

func indexPlaceRow(index map[string]mapsreview.Place, row mapsreview.Place) {
	for _, alias := range mapsreview.PlaceAliases(row.ID, row.URL) {
		index[alias] = row
	}
}

func removePlaceRowAliases(index map[string]mapsreview.Place, row mapsreview.Place) {
	for _, alias := range mapsreview.PlaceAliases(row.ID, row.URL) {
		delete(index, alias)
	}
}

func findExistingRow(index map[string]mapsreview.Place, discovery mapsreview.Discovery) (mapsreview.Place, bool) {
	for _, alias := range mapsreview.PlaceAliases(discovery.ID, discovery.URL) {
		if row, ok := index[alias]; ok {
			return row, true
		}
	}
	return mapsreview.Place{}, false
}
