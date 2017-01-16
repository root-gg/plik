package utils

func Chomp(str string) string {
	if str[len(str)-1] == '\n' {
		str = str[:len(str)-1]
	}
	return str
}
