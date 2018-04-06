# Roadmap

## Config
 * Use TOML instead of JSON to avoid using generic map[string]string

## Schedule
 * Finish net/schedule implementation
 * Move net/schedule to schedule package and add it to base config

## Cache
 * Move net/cache to cache package and add it to base config

## Hystrix
 * Implement hystrix in context

```go
context.Go // Async
context.Do // Sync

context.Go("foo_command", func() error {
	// talk to other services
	return nil
}, func(err error) error {
	// do this when services are down
	return nil
})
```

## Disco
 * Finish serf adapter

## Context package
Refactor the context package

 * Rename ctx -> context
 * Merge app and journey
 * App context should probably be a part of lego package, they share too much logic
 * Rename journey to context? or keep context.Context & context.Journey
 * context.Logger should implement log.Logger

## BG
 * Implement groups with pool of go-routines (e.g. map.update - max 4)

## Admin
 * Add admin package to monitor the app, circuit breakers, drain, ...