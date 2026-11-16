package live

import "errors"

// T023 compile-only declaration: T024 owns validation and hashing.
func verifyManifest(raw []byte, expectedDigest string) (string, error) {
	return "", errors.New("not_implemented")
}
