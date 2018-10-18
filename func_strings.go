package antnet

import (
	"strings"
)

func SplitStr(s string, sep string) []string {
	return strings.Split(s, sep)
}

func StrSplit(s string, sep string) []string {
	return strings.Split(s, sep)
}

func SplitStrN(s string, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

func StrSplitN(s string, sep string, n int) []string {
	return strings.SplitN(s, sep, n)
}

func StrFind(s string, f string) int {
	return strings.Index(s, f)
}

func FindStr(s string, f string) int {
	return strings.Index(s, f)
}

func ReplaceStr(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

func StrReplace(s, old, new string) string {
	return strings.Replace(s, old, new, -1)
}

func ReplaceMultStr(s string, oldnew ...string) string {
	r := strings.NewReplacer(oldnew...)
	return r.Replace(s)
}

func StrReplaceMult(s string, oldnew ...string) string {
	r := strings.NewReplacer(oldnew...)
	return r.Replace(s)
}

func TrimStrSpace(s string) string {
	return strings.TrimSpace(s)
}

func StrTrimSpace(s string) string {
	return strings.TrimSpace(s)
}

func TrimStr(s string, cutset []string) string {
	for _, v := range cutset{
		s = strings.Trim(s, v)
	}
	return s
}

func StrTrim(s string, cutset []string) string {
	return TrimStr(s, cutset)
}

func StrContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func ContainsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}

func JoinStr(a []string, sep string) string {
	return strings.Join(a, sep)
}

func StrJoin(a []string, sep string) string {
	return strings.Join(a, sep)
}

func StrToLower(s string) string {
	return strings.ToLower(s)
}

func ToLowerStr(s string) string {
	return strings.ToLower(s)
}

func StrToUpper(s string) string {
	return strings.ToUpper(s)
}

func ToUpperStr(s string) string {
	return strings.ToUpper(s)
}

func StrTrimRight(s, cutset string) string {
	return strings.TrimRight(s, cutset)
}

func TrimRightStr(s, cutset string) string {
	return strings.TrimRight(s, cutset)
}
