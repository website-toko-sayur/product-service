package searchbuilder

import (
	"encoding/json"
	"errors"

	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
)

func MultiMatchQuery(query string, fields []string) map[string]interface{} {
	return map[string]interface{}{
		"multi_match": map[string]interface{}{
			"query":  query,
			"fields": fields,
		},
	}
}

func TermFilter(field string, value interface{}) map[string]interface{} {
	return map[string]interface{}{
		"term": map[string]interface{}{
			field: value,
		},
	}
}

func RangeFilter(field string, gte interface{}, lte interface{}) map[string]interface{} {

	rangeValue := map[string]interface{}{}

	if gte != nil {
		rangeValue["gte"] = gte
	}

	if lte != nil {
		rangeValue["lte"] = lte
	}

	return map[string]interface{}{
		"range": map[string]interface{}{
			field: rangeValue,
		},
	}
}

func SortQuery(field string, order string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			field: map[string]interface{}{
				"order": order,
			},
		},
	}
}

func ParseOpenSearchError(res *opensearchapi.Response) error {
	if !res.IsError() {
		return nil
	}

	var errResp map[string]interface{}

	if err := json.NewDecoder(res.Body).Decode(&errResp); err != nil {
		return errors.New(res.String())
	}

	errorData, ok := errResp["error"].(map[string]interface{})
	if !ok {
		return errors.New(res.String())
	}

	errType, _ := errorData["type"].(string)

	switch errType {

	case "index_not_found_exception":
		return errors.New("index not found")

	case "mapper_parsing_exception":
		return errors.New("invalid mapping query")

	default:
		return errors.New(res.String())
	}
}
