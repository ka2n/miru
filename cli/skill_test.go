package cli

import (
	"strings"
	"testing"

	"github.com/ka2n/miru/api"
)

func TestRenderSkillGuide(t *testing.T) {
	guide := renderSkillGuide()

	if strings.Contains(guide, "{{") || strings.Contains(guide, "}}") {
		t.Errorf("rendered skill guide still contains unrendered template markers:\n%s", guide)
	}

	for _, sourceType := range api.GetLanguageAliases() {
		if !strings.Contains(guide, sourceType.String()) {
			t.Errorf("rendered skill guide missing source type %q", sourceType.String())
		}
	}
}
