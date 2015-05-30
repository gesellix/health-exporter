package main

import (
	"encoding/json"
	"io/ioutil"
)

type Service struct {
	Uri    string            `json:"uri"`
	Labels map[string]string `json:"labels"`
}

type Config struct {
	Services []Service `json:"services"`
}

func readConfig(file string) (*Config, error) {
	config := &Config{}
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return config, json.Unmarshal(bytes, &config)
}

func (c *Config) collectUniqueLabelNames() []string {
	uniqueLabels := make(map[string]interface{})
	uniqueLabels["uri"] = nil
	for _, service := range c.Services {
		for label, _ := range service.Labels {
			uniqueLabels[label] = nil
		}
	}
	labels := make([]string, len(uniqueLabels))

	i := 0
	for k := range uniqueLabels {
		labels[i] = k
		i += 1
	}
	return labels
}
