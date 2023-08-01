package main

import (
	"fmt"
	"os"
	"strings"
)

var colors = map[string]string{
	"yellow": "\x1B[38;5;220m",
	"blue":   "\x1B[38;5;12m",
	"green":  "\x1B[38;5;46m",
	"red":    "\x1B[38;5;1m",
	"reset":  "\x1B[0m",
}

func preparePrintC(a []any, useColors bool) []any {
	b := make([]any, len(a))
	for i, arg := range a {
		str, isString := arg.(string)

		if !isString {
			b[i] = arg
			continue
		}

		sb := strings.Builder{}
		sb.Grow(len(str))
		for i := 0; i < len(str); i++ {
			r := str[i]
			if r == '\\' {
				sb.WriteByte(r)
				i++
				sb.WriteByte(str[i])
				continue
			}

			if r != '{' {
				sb.WriteByte(r)
				continue
			}

			format := strings.Builder{}
			closingFound := false
			for i++; i < len(str); i++ {
				if str[i] != '}' {
					format.WriteByte(str[i])
				} else {
					closingFound = true
					break
				}
			}

			if !closingFound {
				sb.WriteString(format.String())
			}

			modifications := strings.Split(format.String(), ",")
			for _, mod := range modifications {
				if useColors {
					sb.WriteString(colors[strings.ToLower(strings.Trim(mod, " \t"))])
				}
			}
		}
		b[i] = sb.String()

	}

	return b
}

func printC(a ...any) {
	fmt.Print(preparePrintC(a, colorStdout)...)
}

func printlnC(a ...any) {
	fmt.Println(preparePrintC(a, colorStdout)...)
}

func ePrintC(a ...any) {
	fmt.Fprint(os.Stderr, preparePrintC(a, colorStderr)...)
}

func ePrintlnC(a ...any) {
	fmt.Fprintln(os.Stderr, preparePrintC(a, colorStderr)...)
}
