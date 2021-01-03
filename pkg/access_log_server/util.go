package access_log_server

import (
	"fmt"
	"strconv"
	"time"

	"github.com/seznam/slo-exporter/pkg/stringmap"
)

const (
	nsSep = "."
)

func unmarshallDuration(data map[string]interface{}, key string) (stringmap.StringMap, error) {
	res := stringmap.StringMap{}

	var (
		seconds        float64
		nanos          float64
		seconds_indata bool
		nanos_indata   bool
		ok             bool
	)
	_, seconds_indata = data["seconds"]
	_, nanos_indata = data["nanos"]

	if !seconds_indata && !nanos_indata {
		return res, fmt.Errorf("None of 'seconds' and 'nanos' present in the given data, probably not of expected pbduration data type.")
	}
	if seconds_indata {
		seconds, ok = data["seconds"].(float64)
		if !ok {
			return res, fmt.Errorf("Unable to convert 'seconds' to expected data type.")
		}
	}
	if nanos_indata {
		nanos, ok = data["nanos"].(float64)
		if !ok {
			return res, fmt.Errorf("Unable to convert 'nanos' to expected data type.")
		}
	}

	duration, err := time.ParseDuration(fmt.Sprintf("%fs%fns", seconds, nanos))
	if err != nil {
		return res, err
	}

	res[key] = fmt.Sprint(duration.Nanoseconds()) + "ns"
	return res, nil
}

func unmarshallToMetadata(input interface{}, namespace string) (stringmap.StringMap, error) {
	var err error
	result := stringmap.StringMap{}

	switch v := input.(type) {
	case map[string]interface{}:
		// Assert whether given map is not a representation of Duration
		if result, err = unmarshallDuration(v, namespace); err == nil {
			return result, nil
		}
		// Do a generic map[string]interface unmarshall
		for k, _ := range v {
			res, err := unmarshallToMetadata(v[k], namespace+nsSep+k)
			if err != nil {
				return nil, err
			}
			result = result.Merge(res)

		}
	case []interface{}:
		for k, _ := range v {
			res, err := unmarshallToMetadata(v[k], namespace+nsSep+string(k))
			if err != nil {
				return nil, err
			}
			result = result.Merge(res)
		}
	case float64:
		result[namespace] = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		return stringmap.StringMap{namespace: fmt.Sprint(v)}, nil
	}
	return result, nil
}
