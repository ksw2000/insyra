package stats

import (
	"github.com/HazelnutParadise/insyra"
)

// Skew calculates the skewness(sample) of the DataList.
// Returns the skewness.
// Returns nil if the DataList is empty or the skewness cannot be calculated.
// 錯誤！
func Skew(sample interface{}, method ...string) interface{} {
	d, dLen := insyra.ProcessData(sample)
	d64 := insyra.SliceToF64(d)
	insyra.LogDebug("stats.Skew(): d64: ", d64)
	dl := insyra.NewDataList(d64)
	insyra.LogDebug("stats.Skew(): dl: ", dl)

	methodStr := "pearson"
	if len(method) > 0 {
		methodStr = method[0]
	}
	if len(method) > 1 {
		insyra.LogWarning("stats.Skew(): More than one method specified, using the first one.")
		return nil
	}
	if dLen == 0 {
		insyra.LogWarning("stats.Skew(): DataList is empty, returning nil.")
		return nil
	}

	var result interface{}
	switch methodStr {
	case "pearson":
		result = calculateSkewPearson(dl)
	case "moments":
		result = calculateSkewMoments(dl)
	default:
		insyra.LogWarning("stats.Skew(): Invalid method, returning nil.")
		return nil
	}

	if result == nil {
		insyra.LogWarning("stats.Skew(): Skewness is nil, returning nil.")
		return nil
	}
	resultFloat, ok := result.(float64)
	if !ok {
		insyra.LogWarning("stats.Skew(): Skewness is not a float64, returning nil.")
		return nil
	}
	return resultFloat
}

// ======================== calculation functions ========================
func calculateSkewPearson(sample insyra.IDataList) interface{} {
	mean := sample.Mean()
	median := sample.Median()
	if mean == nil || median == nil {
		insyra.LogWarning("DataList.Skew(): Mean or median is nil, returning nil.")
		return nil
	}
	numerator := 3 * (mean.(float64) - median.(float64))
	denominator := sample.Stdev().(float64)
	if denominator == 0.0 {
		insyra.LogWarning("DataList.Skew(): Denominator is 0, returning nil.")
		return nil
	}

	result := numerator / denominator

	return result
}

func calculateSkewMoments(sample insyra.IDataList) interface{} {
	// todo
	var result float64
	return result
}
