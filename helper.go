package main

import "github.com/hashicorp/nomad/api"

// NewNomadClient ...
func NewNomadClient() (*api.Client, error) {
	return api.NewClient(api.DefaultConfig())
}
