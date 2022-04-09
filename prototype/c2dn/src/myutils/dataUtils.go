package myutils

import (
	"sort"
)

func SumIntSlice(slice []int64) (sliceSum int64) {
	sliceSum = 0
	for _, e := range slice {
		sliceSum += e
	}
	return sliceSum
}

func AvgIntSlice(slice []int64) (sliceAvg float64) {
	sum := SumIntSlice(slice)
	sliceAvg = float64(sum) / float64(len(slice))
	return sliceAvg
}

func MinIntSlice(slice []int64) (minVal int64) {
	minVal = slice[0]
	for _, v := range slice {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func MaxIntSlice(slice []int64) (maxVal int64) {
	maxVal = slice[0]
	for _, v := range slice {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func PercentileIntSlice(slice []int64, percentile float64) (val int64) {
	sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
	return slice[int(float64(len(slice))*percentile/100.0)]
}

func SumFloatSlice(slice []float64) (sliceSum float64) {
	sliceSum = 0
	for _, e := range slice {
		sliceSum += e
	}
	return sliceSum
}

func AvgFloatSlice(slice []float64) (sliceAvg float64) {
	sum := SumFloatSlice(slice)
	sliceAvg = float64(sum) / float64(len(slice))
	return sliceAvg
}
