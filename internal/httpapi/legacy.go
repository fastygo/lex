package httpapi

import (
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/profile/claimvalidation"
)

// legacyProfiles supplies the compatibility profile only for the deprecated
// claim-validation route and its legacy replay bundles.
func (c Config) legacyProfiles() profile.Registry {
	if _, ok := c.Profiles.Default(); ok {
		return c.Profiles
	}
	return profile.MustRegistry(claimvalidation.Profile())
}
