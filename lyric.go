package main

import (
	"regexp"
	"strings"
	"time"
)

type lrcStruct struct {
	t    time.Duration
	word string
}

func loadLyric(lrc string) (bool, []lrcStruct) {
	if len(lrc) == 0 {
		return false, nil
	}

	linesStr := strings.Split(lrc, "\n")

	result := make([]lrcStruct, 0)
	for _, lineStr := range linesStr {
		timeRegex := regexp.MustCompile(`\d{1,2}\:\d{2}\.\d{2,3}`) //11:45.14

		str := timeRegex.FindString(lineStr)
		strMinute := strings.Replace(str, ":", "m", -1)
		strSecond := strings.Replace(strMinute, ".", "s", -1)
		strSecond += "ms"
		newTime, _ := time.ParseDuration(strSecond)

		wordRegex := regexp.MustCompile(`\[.*\]`)
		word := wordRegex.ReplaceAllString(lineStr, "")
		result = append(result, lrcStruct{
			t:    newTime,
			word: word,
		})
	}
	return true, result
}

func getNowLyric(tick time.Duration, lrc []lrcStruct) string {
	for i := 0; i < len(lrc); i++ {
		if i+1 >= len(lrc) {
			return lrc[i].word
		}
		if lrc[i+1].t > tick && lrc[i].t < tick {
			return lrc[i].word
		}
	}
	return ""
}
