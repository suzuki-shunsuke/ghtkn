package config

func (c *Config) SelectApp(key string) *App {
	if c == nil || len(c.Apps) == 0 {
		return nil
	}
	var app *App
	for _, a := range c.Apps {
		if key != "" && a.ID == key {
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
