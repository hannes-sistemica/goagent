# Agent-G Scientific References

Research papers relevant to collective agent intelligence, emergence, and multi-agent coordination.

## Core Hypothesis

> "Intelligence grows not linear with model size, but nonlinear with structure, specialization, and interaction."

These papers provide evidence and methodology for validating this hypothesis.

---

## Multi-Agent Systems & Emergence

### Interpreting Emergent Extreme Events

**Tang, L., Mei, J., Liu, D., Qian, C., Cheng, D. et al. (2026)**
"Interpreting Emergent Extreme Events in Multi-Agent Systems"
arXiv: [2601.20538](https://arxiv.org/abs/2601.20538)

**Summary:** First framework for explaining emergent events in multi-agent systems using Shapley value attribution.

**Key Concepts:**
- Shapley value attribution for agent actions across time
- Three questions: When? Who? What behaviors?
- Multi-dimensional aggregation (time, agent, behavior)
- Tested on economic, financial, social scenarios

**Relevance to Agent-G:**
- Provides rigorous methodology for measuring emergence
- Can quantify "coordination cost vs emergence gain"
- Identifies which agents drive collective outcomes
- Framework for Agent-G experiments

**Implementation Notes:** See [emergence-attribution.md](emergence-attribution.md)

---

### Agentic Fog: Distributed Intelligence

**Akbar, S., Waqas, M., Ullah, R. (2026)**
"Agentic Fog: A Policy-driven Framework for Distributed Intelligence in Fog Computing"
arXiv: [2601.20764](https://arxiv.org/abs/2601.20764)

**Summary:** Policy-driven autonomous agents coordinating via p2p interactions, formalized as potential games.

**Key Concepts:**
- Policy-driven agents (not LLM-powered)
- Potential game formalization for coordination
- Convergence guarantees under asynchronous updates
- P2P shared memory coordination

**Relevance to Agent-G:**
- Validates "structure > scale" hypothesis
- Mathematical foundation for agent coordination
- Shows LLM limitations: "high computational cost, stochastic nature, poor formal analyzability"
- Outperforms greedy heuristics and ILP

**Key Quote:**
> "These tools are not applicable to infrastructure-level systems due to their high computational cost, stochastic nature, and poor formal analyzability."

**Implementation Notes:** See [potential-games.md](potential-games.md)

---

## Modular & Composable AI

### Quantum LEGO Learning

**Qi, J., Yang, C.H.H., Chen, P.Y., Hsieh, M.H., Zenil, H. et al. (2026)**
"Quantum LEGO Learning: A Modular Design Principle for Hybrid Artificial Intelligence"
arXiv: [2601.21780](https://arxiv.org/abs/2601.21780)

**Summary:** Modular, architecture-agnostic learning framework treating components as reusable, composable blocks.

**Key Concepts:**
- Frozen feature block + trainable adaptive module
- Block-wise generalization theory
- Error decomposition: approximation + estimation
- Resource-efficient hybrid learning

**Relevance to Agent-G:**
- Theoretical framework for modular agent composition
- "LEGO" metaphor maps to agent specialization
- Block-wise analysis for understanding agent contributions
- Supports "many small agents" approach

**Key Insight:** Modular composition can match monolithic performance with better interpretability and resource efficiency.

**Implementation Notes:** See [modular-composition.md](modular-composition.md)

---

## Theoretical Foundations

### Game Theory & Multi-Agent Coordination

**Shoham, Y. & Leyton-Brown, K. (2009)**
"Multiagent Systems: Algorithmic, Game-Theoretic, and Logical Foundations"
*Cambridge University Press*
- Foundation for potential game formalization
- Nash equilibrium in multi-agent settings

**Monderer, D. & Shapley, L. (1996)**
"Potential Games"
*Games and Economic Behavior, 14(1), 124-143*
- Original potential games paper
- Convergence guarantees

### Emergence & Complexity

**Holland, J.H. (1998)**
"Emergence: From Chaos to Order"
*Basic Books*
- Emergence from simple rules
- Complex adaptive systems

**Mitchell, M. (2009)**
"Complexity: A Guided Tour"
*Oxford University Press*
- Accessible complexity theory
- Self-organization patterns

### Distributed Systems & Actor Model

**Hewitt, C. (1973)**
"A Universal Modular ACTOR Formalism for Artificial Intelligence"
*IJCAI*
- Original actor model
- Message passing concurrency
- Relevant to agent communication patterns

---

## Agent-G Research Questions

These papers inform our core research questions:

| Question | Relevant Papers |
|----------|-----------------|
| Specialization vs Generalization | Quantum LEGO, Agentic Fog |
| Coordination Cost vs Emergence Gain | Emergent Events (Shapley attribution) |
| Role Stability vs Fluidity | Quantum LEGO (frozen vs adaptive blocks) |
| Decentrality vs Orchestration | Agentic Fog (p2p coordination) |
| Democratization | All - small agents beating large models |

---

## Experiment Design

For Agent-G experiments, consider:

1. **Shapley Attribution** (from Tang et al.)
   - Measure contribution of each agent to collective outcome
   - Track emergence over time
   - Identify critical coordination moments

2. **Potential Game Analysis** (from Akbar et al.)
   - Formalize agent interactions as games
   - Prove convergence properties
   - Analyze stability under failures

3. **Block-wise Evaluation** (from Qi et al.)
   - Decompose system error by component
   - Measure agent specialization value
   - Compare modular vs monolithic architectures
