package utils

import "math"

// 计算均值
func mean(data []float64) float64 {
	sum := 0.0
	for _, value := range data {
		sum += value
	}
	return sum / float64(len(data))
}

// 计算标准差
func StandardDeviation(data []float64) float64 {
	n := len(data)

	if n < 1 {
		return 0 // 标准差需要至少一个数据点
	}

	// 计算平均值
	m := mean(data)

	// 计算方差
	variance := 0.0
	for _, value := range data {
		variance += (value - m) * (value - m)
	}
	variance /= float64(n) // 使用 n 作为分母

	// 计算标准差
	return math.Sqrt(variance)
}
