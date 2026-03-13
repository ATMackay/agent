package documentor

import "errors"

// Config is the base config struct for documentation agent
type Config struct {
	ModelName string
	APIKey    string
	WorkDir   string
}

func (c Config) Validate() error {
	// model name & work dir use defaults
	if c.APIKey == "" {
		return errors.New("missing API key")
	}
	return nil
}
