package assistant

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var numericTokenPattern = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

type groundingResult struct {
	Text     string
	Grounded bool
	Note     string
}

func enforceGrounding(text string, evidence []Evidence) groundingResult {
	text = strings.TrimSpace(text)
	verified := make([]Evidence, 0, len(evidence))
	for _, item := range evidence {
		if item.Verified {
			verified = append(verified, item)
		}
	}

	if len(verified) == 0 {
		if len(numericTokenPattern.FindAllString(text, -1)) > 0 {
			return groundingFallback(nil, "model returned numbers without verified tool evidence")
		}
		return groundingResult{Text: text, Grounded: true}
	}

	for _, item := range verified {
		if !strings.Contains(text, "["+item.ID+"]") {
			return groundingFallback(verified, "model answer omitted an evidence citation")
		}
	}

	allowed := make([]float64, 0)
	for _, item := range verified {
		for _, token := range numericTokenPattern.FindAllString(string(item.Data), -1) {
			if value, err := strconv.ParseFloat(token, 64); err == nil {
				allowed = append(allowed, value)
			}
		}
	}

	for _, token := range numericTokenPattern.FindAllString(text, -1) {
		value, err := strconv.ParseFloat(token, 64)
		if err != nil || !containsNumber(allowed, value) {
			return groundingFallback(verified, "model answer contained a number absent from verified tool evidence")
		}
	}

	return groundingResult{Text: text, Grounded: true}
}

func containsNumber(allowed []float64, candidate float64) bool {
	for _, value := range allowed {
		if math.Abs(value-candidate) <= 1e-9 {
			return true
		}
	}
	return false
}

func groundingFallback(evidence []Evidence, note string) groundingResult {
	if len(evidence) == 0 {
		return groundingResult{
			Text:     "模型输出未通过数值溯源校验。请补充条件后重试，或直接使用下方确定性计算工作台。",
			Grounded: false,
			Note:     note,
		}
	}

	references := make([]string, 0, len(evidence))
	for _, item := range evidence {
		references = append(references, "["+item.ID+"]")
	}
	return groundingResult{
		Text:     fmt.Sprintf("工具调用已完成。模型说明未通过数值溯源校验，请直接查看已验证依据 %s。", strings.Join(references, "、")),
		Grounded: false,
		Note:     note,
	}
}
