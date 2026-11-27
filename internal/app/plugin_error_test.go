package app

import (
	"strings"
	"testing"

	"mygo-music/internal/i18n"
	"mygo-music/internal/plugin"
)

// The errors of an install that the dialog shows are in the listener's language,
// the plugin's name kept as it is.
func TestInstallErrorsInChinese(t *testing.T) {
	i18n.SetLanguage(i18n.Chinese)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })

	if got := installError(&plugin.NameTaken{ID: "qqmusic"}); !strings.Contains(got, "qqmusic") || strings.Contains(got, "a plugin called") {
		t.Errorf("the name taken reads %q", got)
	}
	if got := installError(&plugin.BadRepo{Input: "hhym"}); !strings.Contains(got, "hhym") || strings.Contains(got, "not a GitHub repository") {
		t.Errorf("the bad repository reads %q", got)
	}
	// The error still says what it is, for a log or a test.
	if got := (&plugin.NameTaken{ID: "qqmusic"}).Error(); !strings.Contains(got, "qqmusic") {
		t.Errorf("the error lost its name: %q", got)
	}
}
