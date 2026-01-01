//go:build !embedded

package main

// extractEmbeddedBinaries is a placeholder for the default/downloader version of the app.
// It returns false since there are no embedded binaries.
func (a *App) extractEmbeddedBinaries() (bool, error) {
	return false, nil
}
