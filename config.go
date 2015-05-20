package main

type Service struct {
	Name string `json:"name"`
	Uri  string `json:"uri"`
}

type Config struct {
	Services []Service        `json:"services"`
}
