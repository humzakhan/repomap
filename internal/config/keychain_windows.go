//go:build windows

package config

// Windows keychain integration is not yet implemented.
// Falls back to storing keys in the config file.

func keychainGet(provider string) (string, error) {
	return "", nil
}

func keychainSet(provider string, apiKey string) error {
	return nil
}

func keychainDelete(provider string) error {
	return nil
}
