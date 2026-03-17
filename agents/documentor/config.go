package documentor

import "errors"

// Config is the base config struct for documentation agent
type Config struct {
	WorkDir string
}

func (c Config) Validate() error {
	// ensure workdir is either explicitly set or defaults are set
	// Empty workdir not allowed
	if c.WorkDir == "" {
		return errors.New("empty work dir supplied")
	}
	return nil
}
