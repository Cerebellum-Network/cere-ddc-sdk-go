package config

import "time"

type ClientConfig struct {
	RetryTime         int32
	RetryBackOff      time.Duration
	RequestExpiration time.Duration
	RequestTimeout    time.Duration
	ConnectionTimeout time.Duration
}
