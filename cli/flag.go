package cli

import (
	"github.com/spf13/pflag"
)

type browseTargetFlag struct {
	IsSet bool
	Value string
}

// String implements pflag.Value.
func (s *browseTargetFlag) String() string {
	return s.Value
}

func (s *browseTargetFlag) Set(value string) error {
	s.Value = value
	s.IsSet = true
	return nil
}

func (s *browseTargetFlag) Type() string {
	return "target"
}

var _ pflag.Value = &browseTargetFlag{}
