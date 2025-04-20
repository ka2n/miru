package sourceresolver

import (
	"github.com/ka2n/miru/api/investigator"
	"github.com/ka2n/miru/api/source"
	"github.com/ka2n/miru/api/sourceimpl"
)

// Investigator returns the appropriate investigator for the given SourceType
func Investigator(s source.Type) investigator.SourceInvestigator {
	switch s {
	case source.TypeGitHub:
		return &sourceimpl.GitHubInvestigator{}
	case source.TypeGitLab:
		return &sourceimpl.GitLabInvestigator{}
	case source.TypeNPM:
		return &sourceimpl.NPMInvestigator{}
	case source.TypeGoPkgDev:
		return &sourceimpl.GoPkgDevInvestigator{}
	case source.TypeCratesIO:
		return &sourceimpl.CratesIOInvestigator{}
	case source.TypeRubyGems:
		return &sourceimpl.RubyGemsInvestigator{}
	case source.TypePyPI:
		return &sourceimpl.PyPIInvestigator{}
	case source.TypePackagist:
		return &sourceimpl.PackagistInvestigator{}
	case source.TypeJSR:
		return &sourceimpl.JSRInvestigator{}
	case source.TypeHomepage, source.TypeDocumentation:
		return &sourceimpl.WebsiteInvestigator{Type: s}
	default:
		return nil
	}
}
