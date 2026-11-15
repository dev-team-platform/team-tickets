package config

import (
	"fmt"
	"strconv"
	"strings"
)

// QueueConfig holds queue and retry settings. Delay and jitter values are in milliseconds.
type QueueConfig struct {
	QueueCapacity int
	MaxRetry      int
	RetryDelayMS  []int
	JitterFromMS  int
	JitterToMS    int
}

func loadQueueConfig(prefix string, capacityDefault int, retryDelays string, jitterToDefault int) (QueueConfig, error) {
	capacity, err := intEnv(prefix+"_QUEUE_CAPACITY", capacityDefault)
	if err != nil {
		return QueueConfig{}, err
	}
	if capacity < 1 {
		return QueueConfig{}, fmt.Errorf("%s_QUEUE_CAPACITY must be positive", prefix)
	}
	maxRetry, err := intEnv(prefix+"_MAX_RETRY", 3)
	if err != nil {
		return QueueConfig{}, err
	}
	if maxRetry < 0 {
		return QueueConfig{}, fmt.Errorf("%s_MAX_RETRY cannot be negative", prefix)
	}
	delayName := prefix + "_RETRY_DELAY_MS"
	delays := []int{}
	for _, value := range strings.Split(envOr(delayName, retryDelays), ",") {
		delay, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil || delay < 0 {
			return QueueConfig{}, fmt.Errorf("invalid %s: expected comma-separated non-negative integers", delayName)
		}
		delays = append(delays, delay)
	}
	from, err := intEnv(prefix+"_JITTER_FROM_MS", 1000)
	if err != nil {
		return QueueConfig{}, err
	}
	to, err := intEnv(prefix+"_JITTER_TO_MS", jitterToDefault)
	if err != nil {
		return QueueConfig{}, err
	}
	if from < 0 || to < from {
		return QueueConfig{}, fmt.Errorf("%s_JITTER_FROM_MS and %s_JITTER_TO_MS must be non-negative and ordered", prefix, prefix)
	}
	return QueueConfig{
		QueueCapacity: capacity,
		MaxRetry:      maxRetry,
		RetryDelayMS:  delays,
		JitterFromMS:  from,
		JitterToMS:    to,
	}, nil
}
