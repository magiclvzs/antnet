package antnet

import (
	"math"
	"math/rand"
)

//x^y
func Pow(x, y float64) float64 {
	return math.Pow(x, y)
}

//利用时间取number区间的随机值
func RandNumber(number int) int {
	return rand.Intn(number)
}

//正态分布:根据标准差和期望生成不同的正态分布值,sd标准差,mean期望
func RandNormFloat64(sd, mean int32) float64 {
	return rand.NormFloat64()*float64(sd) + float64(mean)
}

//四舍五入保留n位小数,如保留整数"%.0f",保留3位小数"%.3f"...保留n位小数"%.nf"
func RoundingKeepN(format string, decimal float64) string {
	return Sprintf(format, decimal)
}

//Log10为底x的对数
func Log10(x float64) float64 {
	return math.Log10(x)
}

//abs
func Abs(x float64) float64 {
	return math.Abs(x)
}
