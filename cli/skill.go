package cli

import (
	_ "embed"
	"sort"
	"strings"

	"github.com/ka2n/miru/api"
	"github.com/samber/lo"
)

//go:embed skill.md
var skillGuideTemplate string

const supportedLanguagesTablePlaceholder = "{{SUPPORTED_LANGUAGES_TABLE}}"

// renderSkillGuide renders skill.md with the supported languages table
// filled in from the currently registered language aliases.
func renderSkillGuide() string {
	return strings.Replace(skillGuideTemplate, supportedLanguagesTablePlaceholder, renderSupportedLanguagesTable(), 1)
}

func renderSupportedLanguagesTable() string {
	aliases := api.GetLanguageAliases()
	bySource := lo.GroupBy(lo.Keys(aliases), func(lang string) string {
		return aliases[lang].String()
	})

	sources := lo.Keys(bySource)
	sort.Strings(sources)

	var table strings.Builder
	table.WriteString("| Registry | Aliases |\n")
	table.WriteString("|----------|---------|\n")
	for _, sourceType := range sources {
		aliasList := bySource[sourceType]
		sort.Strings(aliasList)
		table.WriteString("| " + sourceType + " | " + strings.Join(aliasList, ", ") + " |\n")
	}
	table.WriteString("| github.com / gitlab.com | (fallback for unrecognized languages) |\n")
	return table.String()
}
