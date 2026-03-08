//go:build darwin

package config

import (
	"fmt"

	gokeychain "github.com/keybase/go-keychain"
)

const keychainService = "repomap"

// keychainGet retrieves an API key from the macOS Keychain.
func keychainGet(provider string) (string, error) {
	query := gokeychain.NewItem()
	query.SetSecClass(gokeychain.SecClassGenericPassword)
	query.SetService(keychainService)
	query.SetAccount(provider)
	query.SetMatchLimit(gokeychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := gokeychain.QueryItem(query)
	if err != nil {
		return "", fmt.Errorf("querying keychain for %s: %w", provider, err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("no keychain entry for %s", provider)
	}

	return string(results[0].Data), nil
}

// keychainSet stores an API key in the macOS Keychain.
func keychainSet(provider string, apiKey string) error {
	// Delete existing entry first (update = delete + add)
	deleteItem := gokeychain.NewItem()
	deleteItem.SetSecClass(gokeychain.SecClassGenericPassword)
	deleteItem.SetService(keychainService)
	deleteItem.SetAccount(provider)
	_ = gokeychain.DeleteItem(deleteItem) // ignore error if not found

	item := gokeychain.NewItem()
	item.SetSecClass(gokeychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetAccount(provider)
	item.SetData([]byte(apiKey))
	item.SetSynchronizable(gokeychain.SynchronizableNo)
	item.SetAccessible(gokeychain.AccessibleWhenUnlocked)
	item.SetLabel(fmt.Sprintf("Repomap - %s API Key", provider))

	if err := gokeychain.AddItem(item); err != nil {
		return fmt.Errorf("storing keychain entry for %s: %w", provider, err)
	}

	return nil
}

// keychainDelete removes an API key from the macOS Keychain.
func keychainDelete(provider string) error {
	item := gokeychain.NewItem()
	item.SetSecClass(gokeychain.SecClassGenericPassword)
	item.SetService(keychainService)
	item.SetAccount(provider)

	if err := gokeychain.DeleteItem(item); err != nil {
		return fmt.Errorf("deleting keychain entry for %s: %w", provider, err)
	}

	return nil
}
