package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"nuernberg-maps-review-removals/internal/mapsreview"
)

const (
	placesTextSearchURL             = "https://places.googleapis.com/v1/places:searchText"
	placesTextSearchIDOnlyFieldMask = "places.id,nextPageToken"
)

func discoverPlacesAPI(ctx context.Context, args args, dash *mapsreview.Dashboard) ([]mapsreview.Discovery, error) {
	if err := loadDotEnv(".env"); err != nil {
		return nil, err
	}
	apiKey := strings.TrimSpace(os.Getenv("GOOGLE_MAPS_API_KEY"))
	if apiKey == "" {
		return nil, errors.New("GOOGLE_MAPS_API_KEY is required in the environment or .env for --places-api-discovery")
	}

	existing, err := mapsreview.ReadJSON(args.Discovery, []mapsreview.Discovery{})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	completedAPISearches := map[string]bool{}
	discoveries := make([]mapsreview.Discovery, 0, len(existing))
	for _, place := range existing {
		if place.ID == "" || discoverySeen(seen, place) {
			continue
		}
		markDiscoverySeen(seen, place)
		if _, ok := mapsreview.MapsQueryPlaceIDFromURL(place.URL); ok {
			completedAPISearches[apiSearchKey(place.DiscoveredPostcode, place.DiscoveredQuery)] = true
		}
		discoveries = append(discoveries, place)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	stop := false
	for _, postcode := range args.Postcodes {
		for _, query := range args.Queries {
			if args.MaxResults > 0 && len(discoveries) >= args.MaxResults {
				stop = true
				break
			}
			search := fmt.Sprintf("%s %s %s", query, postcode, args.City)
			if args.PlacesAPIPageLimit == 1 && completedAPISearches[apiSearchKey(postcode, query)] {
				dash.Logf("  skipped already saved API search: %s", search)
				continue
			}
			fmt.Printf("\nPlaces API ID-only discover: %s\n", search)

			pageToken := ""
			for page := 0; ; page++ {
				pageSize := 20
				if args.MaxResults > 0 {
					remaining := args.MaxResults - len(discoveries)
					if remaining <= 0 {
						break
					}
					pageSize = min(pageSize, remaining)
				}
				resp, err := placesTextSearch(ctx, client, apiKey, search, pageToken, pageSize)
				if err != nil {
					return nil, err
				}
				for _, place := range resp.Places {
					discovery := discoveryFromAPIPlace(place, postcode, query, search)
					if discovery.ID == "" || discoverySeen(seen, discovery) {
						continue
					}
					markDiscoverySeen(seen, discovery)
					discoveries = append(discoveries, discovery)
				}
				fmt.Printf("\r  places: %d   ", len(discoveries))
				if err := mapsreview.WriteJSON(args.Discovery, discoveries); err != nil {
					return nil, err
				}
				dash.SetDiscoveryCount(len(discoveries))
				if resp.NextPageToken == "" || (args.MaxResults > 0 && len(discoveries) >= args.MaxResults) {
					break
				}
				if page+1 >= args.PlacesAPIPageLimit {
					break
				}
				pageToken = resp.NextPageToken
				time.Sleep(200 * time.Millisecond)
			}
			dash.Logf("  saved %d discoveries", len(discoveries))
			fmt.Printf("\n  saved %d discoveries\n", len(discoveries))
		}
		if stop {
			break
		}
	}
	if args.MaxResults > 0 && len(discoveries) > args.MaxResults {
		discoveries = discoveries[:args.MaxResults]
	}
	return discoveries, nil
}

type placesTextSearchRequest struct {
	TextQuery    string `json:"textQuery"`
	LanguageCode string `json:"languageCode,omitempty"`
	RegionCode   string `json:"regionCode,omitempty"`
	PageSize     int    `json:"pageSize,omitempty"`
	PageToken    string `json:"pageToken,omitempty"`
}

type placesTextSearchResponse struct {
	Places        []placesAPIPlace `json:"places"`
	NextPageToken string           `json:"nextPageToken"`
}

type placesAPIPlace struct {
	ID string `json:"id"`
}

func placesTextSearch(ctx context.Context, client *http.Client, apiKey, textQuery, pageToken string, pageSize int) (placesTextSearchResponse, error) {
	payload := placesTextSearchRequest{
		TextQuery:    textQuery,
		LanguageCode: "de",
		RegionCode:   "DE",
		PageSize:     pageSize,
		PageToken:    pageToken,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return placesTextSearchResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, placesTextSearchURL, bytes.NewReader(body))
	if err != nil {
		return placesTextSearchResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)
	// Keep this ID-only. Google lists Places API Text Search Essentials
	// (IDs Only) with unlimited no-cost usage; displayName/location/URI
	// would move the request to a paid Text Search SKU.
	req.Header.Set("X-Goog-FieldMask", placesTextSearchIDOnlyFieldMask)

	resp, err := client.Do(req)
	if err != nil {
		return placesTextSearchResponse{}, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return placesTextSearchResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return placesTextSearchResponse{}, fmt.Errorf("Places API search failed (%s): %s", resp.Status, strings.TrimSpace(string(respBody)))
	}
	var out placesTextSearchResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return placesTextSearchResponse{}, err
	}
	return out, nil
}

func discoveryFromAPIPlace(place placesAPIPlace, postcode, query, search string) mapsreview.Discovery {
	id := strings.TrimSpace(place.ID)
	return mapsreview.Discovery{
		ID:                 id,
		Name:               search,
		URL:                mapsreview.NormalizeURL(googleMapsSearchURLFromPlaceID(id, search)),
		DiscoveredPostcode: postcode,
		DiscoveredQuery:    query,
	}
}

func apiSearchKey(postcode, query string) string {
	return postcode + "\x00" + query
}

func googleMapsSearchURLFromPlaceID(placeID, name string) string {
	u := url.URL{Scheme: "https", Host: "www.google.com", Path: "/maps/search/"}
	q := u.Query()
	q.Set("api", "1")
	q.Set("query", name)
	q.Set("query_place_id", placeID)
	q.Set("hl", "de")
	u.RawQuery = q.Encode()
	return u.String()
}
