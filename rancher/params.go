package rancher

import (
	"fmt"
	"strings"
)

// SidekickImageParamType descriptor
type SidekickImageParamType struct {
	Name string
	Tag  string
}

// SidekickImageParams define a type as a slice of string
type SidekickImageParams []SidekickImageParamType

// Now, for our new type, implement the two methods of
// the flag.Value interface...
// The first method is String() string
func (i *SidekickImageParams) String() string {
	return fmt.Sprintf("%s", *i)
}

// Set value string
// Value must be like a docker image name[:tag]
func (i *SidekickImageParams) Set(value string) error {
	n := strings.Count(value, ":")
	if n > 1 {
		return fmt.Errorf("Invalid format docker image name [%s]", value)
	}

	p := SidekickImageParamType{
		Name: value,
		Tag:  "",
	}
	if n == 1 {
		parts := strings.Split(value, ":")
		p.Name = parts[0]
		p.Tag = parts[1]
	}
	*i = append(*i, p)
	return nil
}
