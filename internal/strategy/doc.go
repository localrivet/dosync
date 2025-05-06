/*
Copyright © 2024 LocalRivet <github.com/localrivet>

This package provides deployment strategies for updating Docker
containers in a controlled and configurable manner.

# Using the Strategy Package

The strategy package provides several deployment strategies for updating containers:

- OneAtATimeStrategy: Updates replicas one by one
- PercentageStrategy: Updates a percentage of replicas at once
- BlueGreenStrategy: Creates new instances, then switches traffic
- CanaryStrategy: Gradually increases percentage of updated replicas

## Usage Examples

### One-at-a-Time Strategy

```go
// Create a new one-at-a-time strategy

	config := strategy.StrategyConfig{
	    Type:                "one-at-a-time",
	    HealthCheck: health.HealthCheckConfig{
	        Type:    health.TCPHealthCheck,
	        Port:    8080,
	        Timeout: 5 * time.Second,
	    },
	    DelayBetweenUpdates: 5 * time.Second,
	    Timeout:             60 * time.Second,
	    RollbackOnFailure:   true,
	}

// Create the strategy using the factory
strat, err := strategy.NewUpdateStrategy(config, replicaManager, healthChecker)

	if err != nil {
	    // Handle error
	}

// Execute the strategy
err = strat.Execute("my-service", "v2.0.0")

	if err != nil {
	    // Handle error
	}

```

### Percentage-Based Strategy

```go
// Create a new percentage-based strategy

	config := strategy.StrategyConfig{
	    Type:                "percentage",
	    HealthCheck: health.HealthCheckConfig{
	        Type:    health.TCPHealthCheck,
	        Port:    8080,
	        Timeout: 5 * time.Second,
	    },
	    DelayBetweenUpdates: 10 * time.Second,
	    Timeout:             120 * time.Second,
	    RollbackOnFailure:   true,
	    Percentage:          25, // Update 25% at a time
	}

// Create the strategy using the factory
strat, err := strategy.NewUpdateStrategy(config, replicaManager, healthChecker)

	if err != nil {
	    // Handle error
	}

// Execute the strategy
err = strat.Execute("my-service", "v2.0.0")

	if err != nil {
	    // Handle error
	}

```

### Blue/Green Strategy

```go
// Create a new blue/green strategy

	config := strategy.StrategyConfig{
	    Type:              "blue-green",
	    HealthCheck: health.HealthCheckConfig{
	        Type:    health.TCPHealthCheck,
	        Port:    8080,
	        Timeout: 5 * time.Second,
	    },
	    Timeout:           180 * time.Second,
	    RollbackOnFailure: true,
	}

// Create the strategy using the factory
strat, err := strategy.NewUpdateStrategy(config, replicaManager, healthChecker)

	if err != nil {
	    // Handle error
	}

// Execute the strategy
err = strat.Execute("my-service", "v2.0.0")

	if err != nil {
	    // Handle error
	}

```

### Canary Strategy

```go
// Create a new canary strategy

	config := strategy.StrategyConfig{
	    Type:              "canary",
	    HealthCheck: health.HealthCheckConfig{
	        Type:    health.TCPHealthCheck,
	        Port:    8080,
	        Timeout: 5 * time.Second,
	    },
	    Timeout:           300 * time.Second,
	    RollbackOnFailure: true,
	    Percentage:        10, // Initial percentage to deploy
	}

// Create the strategy using the factory
strat, err := strategy.NewUpdateStrategy(config, replicaManager, healthChecker)

	if err != nil {
	    // Handle error
	}

// Execute the strategy
err = strat.Execute("my-service", "v2.0.0")

	if err != nil {
	    // Handle error
	}

```
*/
package strategy
