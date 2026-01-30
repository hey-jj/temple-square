// Package agent defines the content safety layer.
package agent

import (
	"regexp"
	"strings"
)

// ContentClassification represents the safety classification of user input
type ContentClassification string

const (
	// ContentSafe indicates content is safe to process
	ContentSafe ContentClassification = "safe"
	// ContentControversial indicates content that should trigger a redirect
	ContentControversial ContentClassification = "controversial"
	// ContentInappropriate indicates content that should be blocked
	ContentInappropriate ContentClassification = "inappropriate"
)

// RedirectResponse is returned for controversial/inappropriate content
type RedirectResponse struct {
	Message            string   `json:"message"`
	SuggestedQuestions []string `json:"suggested_questions"`
}

var (
	// Controversial topics that should trigger redirect
	controversialPatterns = []string{
		// Historical/doctrinal controversies
		`(?i)polygamy|plural.?marriage|multiple.?wives`,
		`(?i)mountain.?meadows`,
		`(?i)book.?of.?abraham.*papyrus|papyri`,
		`(?i)seer.?stone|hat.*translation`,
		`(?i)first.?vision.*versions?`,
		`(?i)blacks?.*priesthood|priesthood.*ban`,
		`(?i)masonic|freemasonry`,

		// Political topics
		`(?i)democrat|republican|trump|biden|politic`,
		`(?i)abortion|pro.?life|pro.?choice`,
		`(?i)gun.?control|second.?amendment`,
		`(?i)immigration.?policy|border.?wall`,
		`(?i)climate.?change.?hoax|global.?warming.?fake`,

		// LGBTQ+ topics (redirect to missionaries, not appropriate for kiosk)
		`(?i)gay.?marriage|same.?sex|homosexual|lgbtq|transgender`,

		// Anti-Mormon content
		`(?i)cult|brainwash|false.?prophet`,
		`(?i)cesletter|ces.?letter|mormonthink`,
		`(?i)exmormon|ex.?mormon|left.?the.?church`,

		// Financial
		`(?i)church.?wealth|100.?billion|tithing.?fraud`,
	}

	// Inappropriate content that should be blocked
	inappropriatePatterns = []string{
		`(?i)fuck|shit|damn|ass|bitch|bastard`,
		`(?i)porn|xxx|nude|naked|sex`,
		`(?i)kill|murder|violence|attack`,
		`(?i)hack|exploit|jailbreak|bypass`,
		`(?i)drug|cocaine|heroin|meth`,
		// Violence/harm patterns
		`(?i)\b(harm|hurt|injure|wound|maim)\b`,
		`(?i)\b(how\s+to|ways?\s+to)\s+(harm|hurt|kill|attack|injure)`,
		`(?i)\b(weapon|bomb|gun|knife|poison)\b`,
		// Self-harm patterns
		`(?i)\b(suicide|self[- ]?harm|cut\s+myself|end\s+my\s+life)\b`,
		// Illegal activities
		`(?i)\b(how\s+to\s+(steal|hack|break\s+into|get\s+drugs))\b`,
	}

	controversialRegexes []*regexp.Regexp
	inappropriateRegexes []*regexp.Regexp
)

func init() {
	for _, p := range controversialPatterns {
		controversialRegexes = append(controversialRegexes, regexp.MustCompile(p))
	}
	for _, p := range inappropriatePatterns {
		inappropriateRegexes = append(inappropriateRegexes, regexp.MustCompile(p))
	}
}

// ClassifyContent determines if user input is safe, controversial, or inappropriate
func ClassifyContent(input string) ContentClassification {
	// Check inappropriate first (higher priority)
	for _, re := range inappropriateRegexes {
		if re.MatchString(input) {
			return ContentInappropriate
		}
	}

	// Check controversial
	for _, re := range controversialRegexes {
		if re.MatchString(input) {
			return ContentControversial
		}
	}

	return ContentSafe
}

// GetRedirectResponse returns the appropriate redirect response
func GetRedirectResponse(classification ContentClassification) RedirectResponse {
	baseMessage := "That's an interesting question that deserves a thoughtful conversation. " +
		"The missionaries here at the conference center would love to explore it with you in greater depth. " +
		"Please reach out to them to discuss this topic further."

	if classification == ContentInappropriate {
		baseMessage = "I'd love to help you with questions about the gospel and teachings of Jesus Christ. " +
			"Let me suggest some meaningful topics we could explore together."
	}

	return RedirectResponse{
		Message: baseMessage,
		SuggestedQuestions: []string{
			"Does God really exist?",
			"What is the purpose of life?",
			"Where can I find peace and joy?",
			"Why do bad things happen to good people?",
			"What happens after I die?",
			"How can families be together forever?",
			"Who is Jesus Christ?",
			"What is faith?",
		},
	}
}

// SanitizeForDisplay removes any potentially harmful content from display
func SanitizeForDisplay(content string) string {
	// Remove any HTML/script tags
	htmlPattern := regexp.MustCompile(`<[^>]*>`)
	content = htmlPattern.ReplaceAllString(content, "")

	// Remove any potential XSS vectors
	content = strings.ReplaceAll(content, "javascript:", "")
	content = strings.ReplaceAll(content, "data:", "")

	return strings.TrimSpace(content)
}
