package analyzer

import "errors"

// Config is the base config for the analyzer agent.
type Config struct {
	WorkDir string
}

func (c Config) Validate() error {
	if c.WorkDir == "" {
		return errors.New("empty work dir supplied")
	}
	return nil
}
