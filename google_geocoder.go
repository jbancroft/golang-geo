package geo

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type GoogleGeocoder struct{}

var googleZeroResultsError = errors.New("ZERO_RESULTS")

var (
	GoogleApiKey   string
	GoogleClientId string
	GoogleChannel  string
)

// Issues a request to the google geocoding service and forwards the passed in params string
// as a URL-encoded entity.  Returns an array of byes as a result, or an error if one occurs during the process.
func (g *GoogleGeocoder) Request(params string) ([]byte, error) {
	client := &http.Client{}

	fullPath := fmt.Sprintf("/maps/api/geocode/json?sensor=false&%s", params)
	if GoogleApiKey != "" && GoogleClientId != "" {
		fullPath = fmt.Sprintf("%s&client=%s&channel=%s", fullPath, GoogleClientId, GoogleChannel)
		signature, err := sign(fullPath)
		if err != nil {
			return nil, err
		}
		fullPath = fmt.Sprintf("%s&signature=%s", fullPath, signature)
	}
	fullUrl := fmt.Sprintf("http://maps.googleapis.com%s", fullPath)

	// TODO Potentially refactor out from MapQuestGeocoder as well
	req, _ := http.NewRequest("GET", fullUrl, nil)
	resp, requestErr := client.Do(req)

	if requestErr != nil {
		panic(requestErr)
	}

	data, dataReadErr := ioutil.ReadAll(resp.Body)

	if dataReadErr != nil {
		return nil, dataReadErr
	}

	return data, nil
}

func sign(path string) (string, error) {
	rawPrivateKey, err := base64.StdEncoding.DecodeString(GoogleApiKey)
	if err != nil {
		return "", err
	}
	hash := hmac.New(sha1.New, rawPrivateKey)
	_, err = hash.Write([]byte(path))
	if err != nil {
		return "", err
	}
	signature := base64.StdEncoding.EncodeToString(hash.Sum(nil))
	return signature, nil
}

// Geocodes the passed in query string and returns a pointer to a new Point struct.
// Returns an error if the underlying request cannot complete.
func (g *GoogleGeocoder) Geocode(query string) (*Point, error) {
	url_safe_query := url.QueryEscape(query)
	data, err := g.Request(fmt.Sprintf("address=%s", url_safe_query))
	if err != nil {
		return nil, err
	}

	lat, lng, err := g.extractLatLngFromResponse(data)
	if err != nil {
		return nil, err
	}

	p := &Point{lat: lat, lng: lng}

	return p, nil
}

// Extracts the first lat and lng values from a Google Geocoder Response body.
func (g *GoogleGeocoder) extractLatLngFromResponse(data []byte) (float64, float64, error) {
	res := make(map[string][]map[string]map[string]map[string]interface{}, 0)
	json.Unmarshal(data, &res)

	if len(res["results"]) == 0 {
		return 0, 0, googleZeroResultsError
	}

	lat, _ := res["results"][0]["geometry"]["location"]["lat"].(float64)
	lng, _ := res["results"][0]["geometry"]["location"]["lng"].(float64)

	return lat, lng, nil
}

// Reverse geocodes the pointer to a Point struct and returns the first address that matches
// or returns an error if the underlying request cannot complete.
func (g *GoogleGeocoder) ReverseGeocode(p *Point) (string, error) {
	data, err := g.Request(fmt.Sprintf("latlng=%f,%f", p.lat, p.lng))
	if err != nil {
		return "", err
	}

	resStr := g.extractAddressFromResponse(data)

	return resStr, nil
}

// Returns an Address from a Google Geocoder Response body.
func (g *GoogleGeocoder) extractAddressFromResponse(data []byte) string {
	res := make(map[string][]map[string]interface{}, 0)
	json.Unmarshal(data, &res)

	resStr := res["results"][0]["formatted_address"].(string)
	return resStr
}
