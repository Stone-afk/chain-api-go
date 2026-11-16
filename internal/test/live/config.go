package live

import "errors"

// T023 compile-only declaration: T024 owns all configuration behavior.
func loadHarnessConfig(path, alias string, readFile func(string) ([]byte, error), lookupEnv func(string) (string, bool)) (harnessConfig, error) {
	return harnessConfig{}, errors.New("not_implemented")
}
