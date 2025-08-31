package config

// SelectApp selects an app from the configuration based on the provided key.
// Selection priority:
//  1. If key is provided and matches an app ID, returns that app
//  2. If no key match, returns the app marked as default
//  3. If no default app, returns the first app in the list
//  4. Returns nil if config is nil or has no apps
func (c *Config) SelectApp(key string) *App {
	if c == nil || len(c.Apps) == 0 {
		return nil
	}
	var app *App
	for _, a := range c.Apps {
		if key != "" && a.Name == key {
			return a
		}
		if app == nil && a.Default {
			app = a
		}
	}
	if app != nil {
		return app
	}
	return c.Apps[0]
}
