package service

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

// apiURL stores the base api url for generating next and previous page URLs.
var apiURL string

// Contains helpers for all the services
type GetAllResponse struct {

	// The main data of the response
	Data any `json:"data"`

	// Number of records in the response
	Count int `json:"count"`

	// Total number of records in this collection
	TotalCount int64 `json:"total_count"`

	// URL to the next page
	NextPage *string `json:"next_page"`

	// URL to the previous page
	PreviousPage *string `json:"previous_page"`
}

func init() {
	apiURL = os.Getenv("API_URL")
}

// NewGetAllResponse creates a new "GET All" response and returns it as JSON bytes.
func NewGetAllResponse(subpath string, data any, totalCount int64, count, currPage, pageSize int) ([]byte, error) {
	n := float64(totalCount) / float64(pageSize)
	totalPages := int64(math.Ceil(n))
	resp := &GetAllResponse{
		Data:       data,
		Count:      count,
		TotalCount: totalCount,
	}

	// if we're not on the last page, add the next page's URL to the response.
	if totalPages > int64(currPage) {
		nextPageURL := fmt.Sprintf("%s/%s/?pageSize=%d&page=%d", apiURL, subpath, pageSize, currPage+1)
		resp.NextPage = &nextPageURL
	}

	// if we're not on the first page, add the previous page's URL to the response.
	if currPage > 1 && totalPages >= int64(currPage) {
		prevPageURL := fmt.Sprintf("%s/%s/?pageSize=%d&page=%d", apiURL, subpath, pageSize, currPage-1)
		resp.PreviousPage = &prevPageURL
	}

	return json.Marshal(resp)
}

func IsAlphaNumericAndSpaces(fl validator.FieldLevel) bool {
	pattern := "^[A-Za-z0-9 ]*$"
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to compile validation regexp: %v", pattern)
		return false
	}

	// Check if the string matches the pattern
	return re.MatchString(fl.Field().String())
}

func IsState(fl validator.FieldLevel) bool {
	pattern := "^(Alabama|Alaska|Arizona|Arkansas|California|Colorado|Connecticut|Delaware|Florida|Georgia|Hawaii|Idaho|Illinois|Indiana|Iowa|Kansas|Kentucky|Louisiana|Maine|Maryland|Massachusetts|Michigan|Minnesota|Mississippi|Missouri|Montana|Nebraska|Nevada|New Hampshire|New Jersey|New Mexico|New York|North Carolina|North Dakota|Ohio|Oklahoma|Oregon|Pennsylvania|Rhode Island|South Carolina|South Dakota|Tennessee|Texas|Utah|Vermont|Virginia|Washington|West Virginia|Wisconsin|Wyoming)$"
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to compile validation regexp %v", pattern)
		return false
	}

	// Check if the string matches the pattern
	return re.MatchString(fl.Field().String())
}
