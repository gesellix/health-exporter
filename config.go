package main

type Service struct {
	Name   string `json:"name"`
	Uri    string `json:"uri"`
	Labels map[string]string `json:"labels"`
}

type Config struct {
	Services []Service        `json:"services"`
}

func (c *Config) collectLabels() ([]string) {
	uniqueLabels := make(map[string]interface{})
	for _, service := range c.Services {
		for label, _ := range service.Labels {
			uniqueLabels[label] = nil
		}
	}
	uniqueLabels["name"] = nil
	labels := make([]string, len(uniqueLabels))

	i := 0
	for k := range uniqueLabels {
		labels[i] = k
		i += 1
	}
	return labels
}
