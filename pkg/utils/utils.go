package utils

import (
	"encoding/json"

	"github.com/samber/lo"
)

func MergeStructToMap(structs []interface{}) (map[string]interface{}, error) {
	maps := lo.Map(structs, func(item interface{}, _ int) map[string]interface{} {
		var data map[string]interface{}
		if bytes, err := json.Marshal(item); err == nil {
			if err := json.Unmarshal(bytes, &data); err != nil {
				return nil
			}
		}
		return data
	})
	maps = lo.Filter(maps, func(item map[string]interface{}, index int) bool {
		return item != nil
	})
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result, nil
}
