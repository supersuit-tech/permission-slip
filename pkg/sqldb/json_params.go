package sqldb

import (
	"encoding/json"
	"math"
)

// CoerceJSONParamValue normalizes values produced by encoding/json.Unmarshal
// (e.g. whole-number float64 → int64) before passing them to database/sql.
func CoerceJSONParamValue(v interface{}) interface{} {
	switch x := v.(type) {
	case float64:
		if !math.IsNaN(x) && !math.IsInf(x, 0) && x == math.Trunc(x) &&
			x >= float64(math.MinInt64) && x < float64(math.MaxInt64)+1 {
			return int64(x)
		}
		return x
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, _ := x.Float64()
		return f
	case []interface{}:
		out := make([]interface{}, len(x))
		for i := range x {
			out[i] = CoerceJSONParamValue(x[i])
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			out[k] = CoerceJSONParamValue(val)
		}
		return out
	default:
		return v
	}
}
