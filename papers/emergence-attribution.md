# Emergence Attribution with Shapley Values

**Paper:** Tang et al. (2026) - arXiv:2601.20538

## Overview

This paper provides the first rigorous framework for explaining emergent events in multi-agent systems. Directly applicable to Agent-G experiments.

## The Three Questions

The framework answers:

1. **When** does the emergent event originate?
2. **Who** (which agents) drives it?
3. **What behaviors** contribute to it?

## Shapley Value for Agent Actions

### Background

Shapley value fairly attributes collective outcomes to individual contributions:

```
φᵢ = Σ [|S|!(n-|S|-1)!/n!] × [v(S∪{i}) - v(S)]
```

Where:
- `φᵢ` = attribution score for agent i
- `v(S)` = value function for coalition S
- Sum over all possible coalitions

### Adaptation for Agent Actions

The paper adapts Shapley to attribute emergent events to:
- Each action
- At each time step
- By each agent

```go
// Conceptual implementation for Agent-G
type ActionAttribution struct {
    AgentID    string
    TimeStep   int
    Action     string
    ShapleyValue float64  // Influence on emergent outcome
}

func ComputeShapleyAttribution(
    history []AgentAction,
    outcome EmergentEvent,
) []ActionAttribution {
    // For each action in history:
    // 1. Compute marginal contribution to outcome
    // 2. Average over all possible coalitions/orderings
    // 3. Assign Shapley value as attribution score
}
```

## Multi-Dimensional Aggregation

### Time Dimension

When does emergence begin?

```go
type TemporalRisk struct {
    TimeStep      int
    CumulativeRisk float64
}

func AggregateByTime(attrs []ActionAttribution) []TemporalRisk {
    // Sum Shapley values by time step
    // Identifies when critical mass reached
}
```

### Agent Dimension

Which agents are most responsible?

```go
type AgentRisk struct {
    AgentID      string
    TotalRisk    float64
    ActionCount  int
}

func AggregateByAgent(attrs []ActionAttribution) []AgentRisk {
    // Sum Shapley values by agent
    // Identifies key drivers of emergence
}
```

### Behavior Dimension

What action types matter?

```go
type BehaviorRisk struct {
    ActionType   string
    TotalRisk    float64
    Frequency    int
}

func AggregateByBehavior(attrs []ActionAttribution) []BehaviorRisk {
    // Sum Shapley values by action type
    // Identifies critical behaviors
}
```

## Application to Agent-G Experiments

### Experiment: Specialization vs Generalization

Question: When are multiple specialized agents better than one generalist?

**Using Shapley Attribution:**
1. Run task with N specialists
2. Compute Shapley values for each agent's actions
3. Measure emergence of collective intelligence
4. Compare to single generalist baseline

```go
func CompareSpecialization(
    specialists []Agent,
    generalist Agent,
    task Task,
) SpecializationResult {
    // Run both configurations
    specResult := RunWithShapleyTracking(specialists, task)
    genResult := RunWithShapleyTracking([]Agent{generalist}, task)

    return SpecializationResult{
        SpecialistPerformance: specResult.Outcome,
        GeneralistPerformance: genResult.Outcome,
        EmergenceGain: specResult.EmergenceMetric - genResult.EmergenceMetric,
        CoordinationCost: specResult.CoordinationOverhead,
        ShapleyAnalysis: specResult.Attributions,
    }
}
```

### Experiment: Coordination Cost vs Emergence Gain

Question: At what point does agent interaction cost exceed emergence benefit?

**Metrics from Paper:**
- Coordination overhead (message count, latency)
- Emergence metric (collective capability beyond sum of parts)
- Shapley-based contribution inequality (Gini coefficient of attributions)

### Experiment: Critical Agents

Question: Which agents are essential vs redundant?

**Using Agent Aggregation:**
- High Shapley sum = critical agent
- Low Shapley sum = potentially redundant
- Negative Shapley = counterproductive agent

## Implementation Checklist

- [ ] Implement Shapley value computation for agent actions
- [ ] Add time-step tracking to agent action logs
- [ ] Create aggregation functions (time, agent, behavior)
- [ ] Define emergence metrics for Agent-G tasks
- [ ] Build visualization for attribution analysis
- [ ] Integrate into experiment framework

## Key Insight

> "The interactions within these systems often lead to extreme events whose origins remain obscured by the black box of emergence."

This framework opens the black box - making emergence interpretable and measurable.

## Citation

```bibtex
@article{tang2026interpreting,
  title={Interpreting Emergent Extreme Events in Multi-Agent Systems},
  author={Tang, Ling and Mei, Jilin and Liu, Dongrui and Qian, Chen and Cheng, Dawei},
  journal={arXiv preprint arXiv:2601.20538},
  year={2026}
}
```
