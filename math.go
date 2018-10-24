package antnet

import (
	"math"
	"math/rand"
)

//x^y
func Pow(x, y float64) float64 {
	return math.Pow(x, y)
}

//返回x的二次方根
func Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

//取0-number区间的随机值
func RandNumber(number int) int {
	if(number == 0) {
		return 0
	}
	return rand.Intn(number)
}

//min和max之间的随机数
func RandNumBetween(min, max int) int {
	if min == max {
		return min
	}
	if min > max {
		min, max = max, min
	}
	return rand.Intn(max-min) + min
}

//正态分布:根据标准差和期望生成不同的正态分布值,sd标准差,mean期望
func RandNorm64(sd, mean int32) float64 {
	return rand.NormFloat64()*float64(sd) + float64(mean)
}

//正态分布,在一定范围内
func RandNormInt32(min, max, sd, mean int32) int32 {
	result := int32(Atoi(RoundStr("%0.0f", RandNorm64(sd, mean))))
	if result < min {
		return min
	}
	if result > max {
		return max
	}
	return result
}

//四舍五入保留n位小数,如保留整数"%.0f",保留3位小数"%.3f"...保留n位小数"%.nf"
func RoundStr(format string, decimal float64) string {
	return Sprintf(format, decimal)
}

//四舍五入保留n位小数
func Round(decimal float64, w int) float32 {
	format := "%." + Itoa(w) + "f"
	return Atof(Sprintf(format, decimal))
}

//四舍五入保留n位小数
func Round64(decimal float64, w int) float64 {
	format := "%." + Itoa(w) + "f"
	return Atof64(Sprintf(format, decimal))
}

//Log10为底x的对数
func Log10(x float64) float64 {
	return math.Log10(x)
}

//abs
func Abs(x float64) float64 {
	return math.Abs(x)
}

//int32
func MaxInt32() int32 {
	return math.MaxInt32
}
