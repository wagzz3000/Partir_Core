# Partir Core

**A framework for diagnosing, treating, and preventing semantic and systemic drift in persistent AI environments.**

**TL;DR**

Partir addresses the maintenance problem created when AI systems persist across long-running agents, subagents, models, memory stores, tools, and changing environments. Its central distinction is between **semantic drift** (the system establishes or maintains an incorrect meaning, relationship, classification, or representation) and **systemic drift** (the conditions, information, models, dependencies, constraints, or environment surrounding inference become incorrect, incomplete, stale, or misaligned). Partir proposes diagnostics, graduated intervention, graph repair, validation gates, maintenance rotation, and preventative propagation for managing those failures over time.

The repository contains the architecture, research, patent material, technical specifications, supporting assets, and an early-stage Go prototype. The full conceptual model is developed below; the prototype should be understood as an implementation reference rather than a finished production platform.

### Repository at a Glance

```text
├── docs/architecture/      ← Patent application + system diagrams
├── docs/design/            ← Technical specifications
├── docs/guides/            ← Prototype setup and usage guides
├── docs/reference/         ← API documentation
├── docs/research/          ← Research and supporting papers
├── docs/assets/            ← Screenshots and prototype material
└── prototype/              ← Early-stage Go implementation
```

Partir is an architectural and research project focused on a problem that becomes increasingly important as AI systems move from short-lived inference toward long-running, persistent operation:

> **How do you keep an AI system reliable when its state, models, inputs, tools, and environment all change over time?**

Partir does not treat this as a problem limited to one persistent agent instance. The object of concern is the **persistent agent environment**: the computational environment through which memory, information, inference, models, tools, agents, subagents, and external conditions continue to interact.

The project proposes an architecture for observing that environment, diagnosing emerging failures, applying targeted interventions, validating those interventions, and preventing known failure modes from propagating through the system.

---

# 1. The Problem: Persistent AI Environments

Most AI systems are evaluated primarily at the level of individual interactions.

A prompt goes in. An answer comes out. The answer is evaluated.

That model becomes increasingly inadequate when an AI system operates continuously for weeks, months, or years.

A persistent environment accumulates state.

That state can include:

- memories
- inferred relationships
- semantic associations
- behavioral patterns
- task state
- generated artifacts
- tool knowledge
- organizational requirements
- technical decisions
- historical context
- model assumptions
- environmental observations

At the same time, the environment around the system continues to change.

Software changes.

Requirements change.

Law changes.

Terminology changes.

Models change.

Tools change.

External conditions change.

New information arrives while old information becomes stale.

The result is a system in which reliability is no longer only a question of whether a single inference is correct.

It becomes a question of whether the **environment remains coherent over time**.

---

## 1.1 Persistence Does Not Require One Persistent Agent

Partir is therefore not limited to a single agent instance remaining alive indefinitely.

A persistent environment can hand work between:

- agents
- subagents
- model instances
- workers
- processes
- services

The environment remains persistent even when the individual agent does not.

This creates another source of failure: **handoff degradation**.

A receiving agent may not receive the complete state of the previous agent. It may instead receive:

- a summary
- selected memories
- retrieved documents
- a task description
- compressed context
- structured state
- generated notes
- tool outputs
- a knowledge graph

Every transformation involved in that handoff can lose information.

Summaries can be incomplete.

Summaries can be inaccurate.

Important context can be omitted because it appeared irrelevant to the summarizing agent.

An ambiguous statement can become an apparently definitive statement.

A conditional relationship can become an unconditional one.

A low-confidence inference can become indistinguishable from an established fact.

The receiving agent then begins operating from a representation that is already different from the state of the previous agent.

The receiving agent also brings its own learned assumptions, model-level drift, and potentially stale information.

The result can compound across repeated handoffs:

**Agent A → summary → Agent B → reinterpretation → summary → Agent C → reinterpretation**

Each individual agent may appear to operate normally.

The environment as a whole can nevertheless drift substantially.

This is why Partir's unit of concern is the **persistent computational environment**, not necessarily the individual agent.

---

## 1.2 Long-Running Sensor Systems

The same problem exists outside conversational AI.

Consider an autonomous commercial semi-truck.

Commercial trucks can remain on the road for many hours and potentially across multiple days of operation. An autonomous driving system therefore has to process a continuous stream of computationally expensive camera and sensor data while maintaining enough contextual state to make decisions.

The challenge is not simply whether an individual inference is correct.

The system operates continuously inside a changing environment.

Hours of sensor observations can create enormous amounts of transient and persistent computational state. The system has to decide what to retain, what to discard, what to summarize, what to trust, and what remains relevant to future decisions.

Every one of those transformations creates an opportunity for drift.

Environmental conditions introduce another dimension.

Changing cloud cover, for example, can alter illumination and therefore the appearance of colors, contrast, shadows, and surfaces in camera data.

A visual system that interprets those changes incorrectly could turn an environmental change into a semantic interpretation error.

The point is not that changing cloud cover necessarily causes a specific failure.

The point is that a long-running autonomous system must remain robust while the statistical characteristics of its inputs change over time.

A system operating under one lighting distribution may later encounter a materially different distribution.

That is a systemic change in the environment.

If the system's internal representation does not adapt appropriately, the systemic change can become a semantic failure.

In a safety-critical system such as autonomous transportation, even a relatively small perceptual error can have consequences far beyond the apparent size of the original error.

This illustrates why **error magnitude and environmental context must be considered together**.

---

## 1.3 Persistence Exists at Multiple Layers

Persistent state can exist at several layers of an AI environment:

- **Agent state** — current context, memory, and behavioral state
- **Fleet state** — information shared between multiple agents
- **Graph state** — derived semantic relationships
- **Historical state** — the underlying timeline or ledger
- **Task state** — requirements, decisions, specifications, and work products
- **Model state** — learned behavior and inherited assumptions
- **Tool state** — APIs, software versions, databases, and external services
- **Environmental state** — external conditions against which the system operates

A maintenance system that monitors only one agent instance therefore sees only part of the problem.

Partir is concerned with the integrity of the entire persistent environment and the transformations that move information through it.

---

# 2. Drift

The central distinction in Partir is between **systemic drift** and **semantic drift**.

They are related, but they occur at different stages of the system's lifecycle.

The simplest way to understand the relationship is:

```text
SYSTEMIC CONDITIONS
        ↓
     INFERENCE
        ↓
 SEMANTIC STATE
        ↓
   PERSISTENCE
        ↓
FUTURE SYSTEMIC INPUT
        ↓
     INFERENCE
        ↓
 SEMANTIC STATE
        ↓
       ...
```

**Systemic conditions** are the environment, information, models, dependencies, constraints, and other conditions under which inference takes place.

**Semantic state** is what the system concludes, represents, associates, or otherwise establishes as meaning as a result of that inference.

A systemic problem can therefore produce a semantic problem.

A semantic problem can then become part of the systemic input to a future inference.

This feedback loop is one of the central failure mechanisms Partir is designed to address.

---

## 2.1 Systemic Drift

Systemic drift is drift in the **conditions surrounding inference**.

Those conditions can include:

- source data
- training data
- retrieved information
- agent handoffs
- summaries
- model behavior
- software dependencies
- organizational requirements
- technical specifications
- laws and regulations
- external services
- physical environments
- sensor conditions

A systemic defect does not necessarily mean that the inference process itself is malfunctioning.

The system may reason perfectly according to what it was given.

The problem is that **what it was given is no longer correct, complete, appropriate, or aligned with the environment in which the system is operating**.

### Flawed training data

Consider an LLM trained on a large corpus of publicly available source code.

That corpus contains a mixture of:

- current code
- obsolete code
- abandoned projects
- incomplete implementations
- incorrect implementations
- duplicated implementations
- contradictory approaches
- code written for obsolete versions
- code written under assumptions that no longer hold

The presence of obsolete or incorrect material in the training corpus is a **systemic condition**.

The model may absorb those historical patterns into its learned behavior.

Suppose an organization later asks the model to implement a feature using a current API.

The model has learned thousands of examples using an obsolete version of that API.

It may therefore generate an implementation based on the obsolete pattern.

The failure unfolds across two stages:

```text
Flawed / obsolete training data
        ↓
Learned model assumptions
        ↓
Inference against current task
        ↓
Incorrect implementation or interpretation
```

The flawed training data and inherited model assumptions represent **systemic drift**.

The incorrect interpretation or generated implementation is a **semantic failure**.

This distinction matters because correcting the output does not necessarily correct the systemic source of the problem.

The system may continue producing the same semantic failure because the underlying model or information environment remains unchanged.

---

## 2.2 Semantic Drift

Semantic drift is drift in the **meaning, relationship, classification, or representation produced and maintained by the system**.

A useful shorthand is:

> **The system has established the wrong meaning.**

A semantic failure can occur even when the information entering inference is correct.

For example, suppose a codebase explicitly defines:

```text
User = authenticated account
```

The source information is correct.

If the agent's knowledge graph represents:

```text
User = customer
```

the semantic representation is wrong.

The system has received valid information but established an incorrect meaning.

Likewise, a translation system may receive a perfectly valid phrase and still assign the wrong contextual meaning to a word.

The important distinction is therefore:

> **Systemic drift concerns the conditions under which meaning is inferred. Semantic drift concerns the meaning that is inferred and maintained.**

---

## 2.3 The Two Failure Modes Can Compound

The two categories are not mutually exclusive.

They can form a feedback loop.

For example:

```text
Outdated specification
        ↓
SYSTEMIC DRIFT
        ↓
Inference
        ↓
Incorrect relationship
        ↓
SEMANTIC DRIFT
        ↓
Relationship stored in memory
        ↓
Future agent retrieves relationship
        ↓
SYSTEMIC INPUT
        ↓
Inference
        ↓
Additional semantic error
```

The original failure was systemic.

The resulting persistent state became semantic.

Once stored, that semantic error became part of the systemic input to the next inference.

This is why simply restarting an individual agent may not solve a persistent-environment problem.

The problematic state may have survived the restart in:

- memory
- the knowledge graph
- task state
- summaries
- shared fleet state
- generated documentation
- model configuration
- external databases

The environment can therefore continue reproducing the same failure even after the original agent instance is gone.

---

## 2.4 Agent and Subagent Handoffs

Handoffs provide a particularly clear example of how the categories interact.

Suppose Agent A performs a task and creates a summary for Agent B.

The summary is incomplete.

That is a **systemic defect in the information entering Agent B's inference**.

Agent B then interprets the incomplete summary and establishes an incorrect relationship.

That is a **semantic failure**.

The resulting relationship is then passed to Agent C as part of its context.

It has now become another systemic input.

The cycle can repeat:

```text
Agent A
   ↓
Incomplete summary
   ↓
SYSTEMIC
   ↓
Agent B inference
   ↓
Incorrect interpretation
   ↓
SEMANTIC
   ↓
Persistent state
   ↓
Agent C input
   ↓
SYSTEMIC
   ↓
Agent C inference
   ↓
...
```

A different LLM at each stage can increase the problem because different models may have different learned associations, contextual interpretations, and assumptions.

This is why Partir treats the transformations between agents as part of the maintenance problem.

---

## 2.5 Changing the Graph-Producing LLM

The same reasoning applies to the LLM responsible for constructing a knowledge graph.

Changing that model is itself a **systemic change to the inference pipeline**.

Suppose an environment has operated for months with one model responsible for extracting entities and relationships from its timeline.

Replacing that model with another can change:

- entity extraction
- relationship extraction
- entity resolution
- contextual interpretation
- confidence estimates
- synonym handling
- graph topology
- classification boundaries
- assumptions about meaningful relationships

The underlying timeline may remain unchanged while the derived graph changes substantially.

The model change is therefore a systemic change.

If the new model then constructs an incorrect relationship, the resulting relationship is a semantic failure.

```text
Model change
    ↓
SYSTEMIC CHANGE
    ↓
Graph-construction inference
    ↓
Incorrect relationship
    ↓
SEMANTIC FAILURE
```

A model substitution should therefore be treated as a potentially significant system change and subjected to validation and regression testing.

---

## 2.6 Maintenance Models Can Introduce Both

The same problem applies to models responsible for diagnosing or repairing the graph.

A separate LLM may be used to recommend a maintenance procedure based on detected problems.

That maintenance model can encounter systemic problems on its own input side:

- stale documentation
- incomplete graph state
- missing context
- outdated specifications
- incorrect source information

Those are systemic conditions.

It can then produce a semantic failure by misunderstanding a node, relationship, definition, or context and recommending the wrong intervention.

If it recommends an incorrect graph modification, that intervention can introduce additional semantic drift into persistent state.

The distinction therefore remains:

```text
Bad maintenance input
        ↓
SYSTEMIC
        ↓
Maintenance inference
        ↓
Wrong interpretation / recommendation
        ↓
SEMANTIC
        ↓
Structural modification
        ↓
New persistent state
```

Using a different LLM for maintenance does not eliminate this risk.

It may introduce additional systemic differences because the maintenance model has its own training history, learned associations, terminology, and interpretation behavior.

> **The component responsible for maintaining a representation is itself part of the representation's threat model.**

A maintenance model cannot automatically be treated as an authoritative judge simply because it is performing maintenance.

Its recommendations must themselves be evaluated.

---

## 2.7 Systemic Drift at the Model Level

Systemic drift can exist before an agent is deployed.

Large language models are trained on enormous quantities of historical information.

Public code is an especially useful example because it contains a mixture of current and historical practices.

The model therefore begins operation with an inherited information environment that may already contain:

- obsolete assumptions
- incomplete information
- incorrect examples
- contradictory practices
- outdated APIs
- historical terminology
- abandoned approaches

Deployment then adds another layer.

An organization may change:

- its architecture
- coding standards
- security requirements
- APIs
- dependencies
- terminology
- operational procedures
- legal requirements

The model's learned information does not automatically update when those things change.

A persistent environment therefore operates at the intersection of several timelines:

**model knowledge → organizational state → software state → legal state → current task**

The longer those timelines diverge, the greater the opportunity for systemic drift.

---

## 2.8 Distillation as a Potential Drift Vector

Model distillation presents another potential systemic-drift mechanism.

A distilled model may inherit behavioral and representational characteristics from its parent model.

If the parent model contains systemic drift, some of that drift may be inherited by the distilled model.

Additional training data can then introduce further assumptions, biases, or errors.

Conceptually:

```text
Parent model systemic drift
        ↓
Inherited systemic drift
        ↓
Additional training data
        ↓
Additional systemic drift
        ↓
Distilled model
```

This is a **hypothesis within the Partir research model**, not an established empirical result.

The logic warrants investigation, but the repository should not present the mechanism as demonstrated without supporting data.

---

# 3. Not All Wrongness Is Equal

Partir does not treat correctness as a simple binary property.

An output or relationship can be wrong without all wrongness being equivalent.

A useful illustration comes from a discussion in *The Big Bang Theory* in which Sheldon treats a proposition as simply correct or wrong, while Stuart points out that there can be degrees of wrongness.

Consider a tomato.

Confusing a tomato's botanical classification with its common culinary classification is a relatively small error.

Confusing a tomato with a suspension bridge is an enormous error.

Both propositions are wrong.

They are not equally wrong.

For an AI maintenance system, this distinction matters.

A diagnostic system should not merely ask:

> **Is this relationship incorrect?**

It should also ask:

> **How incorrect is it, and what is the consequence of being wrong here?**

An error involving a weak, low-impact association may require little or no intervention.

An error involving a highly consequential semantic relationship may require immediate isolation, deeper diagnosis, and structural repair.

This introduces the idea of **semantic gravity**.

---

# 4. Semantic Gravity and the Three-Dimensional Graph

A conventional graph represents nodes and relationships.

Partir explores a model in which relationships also have a third dimension representing their relative significance, magnitude, or **semantic gravity**.

Conceptually:

- **X/Y dimensions** — graph topology and relationships
- **Z dimension** — relative semantic gravity, significance, or error magnitude

The exact mathematical formulation remains an area for further development.

The important architectural point is that **graph topology alone is insufficient**.

Two incorrect relationships can have radically different consequences.

A system therefore needs to understand not only:

**whether a relationship is wrong**

but also:

**how much the wrongness matters.**

---

## 4.1 Frequency Is Not the Same as Gravity

Consider an AI translation system.

The letter **A** appears in an enormous number of words and sentences.

As a graph node, it could therefore have an extremely large number of connections.

Yet an individual occurrence of A provides relatively little predictive information about a particular word or phrase because the letter is so common.

Its graph presence is large.

Its individual semantic gravity is comparatively shallow.

Now consider **Z**.

It is much less common.

An occurrence of Z therefore provides more information about the possible identity of a word than a generic occurrence of A.

Its graph may be smaller, but its relative informational significance can be greater.

This illustrates why:

**node size ≠ connection count ≠ semantic gravity**

A highly connected node is not automatically a highly meaningful node.

---

## 4.2 Context Changes Meaning

The problem becomes more complicated when an AI system processes language through multiple representations.

A translation system does not necessarily operate only on written text. It can also process sound, phonetics, pronunciation, inflection, surrounding words, syntax, cultural context, and transliteration.

These representations can create relationships that are useful in one context but misleading in another.

For example, phonetic or transliteration similarity can connect otherwise unrelated concepts. Inflection can also fundamentally change meaning:

- **papa**
- **papá**

Likewise, distinctions such as:

- **chi / qi**
- **colon** as a medical term versus **colon** as the punctuation symbol `:`

can matter enormously depending on context.

The important point is not that any particular pair necessarily produces a specific graph topology. It is that a persistent semantic system must account for **context-dependent meaning across multiple representational layers**.

A relationship can be linguistically valid, phonetically valid, visually similar, historically associated, or contextually valid while still being invalid for the inference currently being performed.

This is why semantic drift cannot be adequately monitored by checking individual graph edges in isolation.

# 5. Drift Can Enter Anywhere in the Pipeline

A persistent AI environment is not one model. It is a pipeline of transformations:

**source data → primary agent → graph construction → derived graph → diagnostics → maintenance recommendation → intervention → validation → updated state**

Every model and transformation in that chain can introduce drift — either systemic (degraded input conditions) or semantic (incorrect derived meaning), or both.

Sections 2.4 through 2.8 develop specific examples of this in detail: agent handoffs, model substitution in graph construction, maintenance models introducing new errors, inherited model drift, and distillation as a propagation vector.

The architectural implication is straightforward:

> **No component in the pipeline is automatically trustworthy simply because of its role.**

A graph-producing model can introduce drift. A maintenance model can introduce drift. A summarizing agent can introduce drift. A distilled model can inherit drift.

The maintenance architecture must therefore treat every transformation as a potential drift source and subject each to the same validation discipline applied to production output.

---

# 6. Provenance Matters

If drift can enter through any transformation in the pipeline, provenance becomes important to maintenance.

A derived relationship should ideally retain enough information to answer questions such as:

- Which model produced this relationship?
- Which version of that model?
- What source material was used?
- What transformation produced the relationship?
- What confidence or evidence supported it?
- Which model recommended a subsequent intervention?
- What intervention was performed?
- What validation was performed afterward?
- Did the resulting behavior improve?

Without this information, diagnosis becomes substantially harder.

A system may know that a relationship changed without knowing whether the change originated in:

- the source data
- the model
- the graph-construction process
- an agent handoff
- a maintenance recommendation
- a structural intervention
- an environmental change

Provenance therefore becomes part of the maintenance problem itself.

---

# 7. Source of Truth and Derived State

A graph is not necessarily safe as the authoritative source of truth.

Once semantic relationships have been inferred, those relationships can themselves be contaminated.

A graph can faithfully preserve an incorrect interpretation.

Partir therefore separates two related representations.

## Timeline / Ledger

The timeline represents what actually happened.

It is the source-of-truth historical record.

The purpose is to preserve the underlying events and information independently of whether later semantic interpretation is correct.

## Derived Knowledge Graph

The knowledge graph is derived from that history.

It is useful for reasoning, relationship analysis, diagnostics, and inference.

But because it is derived, it can be reconstructed, migrated, pruned, or repaired.

This distinction makes targeted maintenance possible.

If a semantic interpretation becomes invalid, Partir does not necessarily need to erase the historical record.

Instead, it can:

1. Identify the affected semantic structure.
2. Determine which relationships depend on the invalid meaning.
3. Remove or quarantine affected relationships.
4. Reconstruct the derived structure.
5. Validate the resulting graph.
6. Compare behavior before and after the intervention.

This is the conceptual basis of **graph surgery**.

---

# 8. What Partir Does

Partir treats maintenance as a controlled lifecycle rather than an emergency restart procedure.

The core lifecycle is:

**Detect → Diagnose → Treat → Validate → Return to Production → Prevent**

---

## 8.1 Diagnostics

Partir's diagnostic layer is intended to identify meaningful changes in system behavior before they become obvious production failures.

Traditional observability asks:

- Is the process alive?
- Is latency increasing?
- Are requests failing?
- Is the database available?
- Are resources being exhausted?

Those measurements remain useful.

They do not answer another class of question:

> **Is the persistent AI environment beginning to behave differently?**

Partir therefore considers behavioral and semantic indicators alongside conventional infrastructure telemetry.

The diagnostic approach includes concepts such as:

- pattern detection
- graph analysis
- probe-based testing
- semantic consistency
- relationship validity
- information freshness
- source reliability
- behavioral patterns
- intervention history
- regression behavior
- error magnitude

The goal is to distinguish:

**normal variation → meaningful drift → high-impact failure requiring intervention**

---

## 8.2 Treatment

Not every problem deserves the same intervention.

Partir therefore conceptualizes treatment as graduated.

### Parameter-Level Intervention

Temporary changes to operational parameters can provide a fast, targeted, reversible intervention.

The conceptual analogy used during development was medication.

### Structured Interaction

Some problems may require deliberate interaction with the agent to reshape how it relates to particular memories or concepts.

The conceptual analogy was therapy.

### Rhythmic Activation

Another experimental concept involves controlled rhythmic activation of different model components.

This was inspired by EMDR and the broader question of whether patterned activation could help disrupt persistent associations.

This is an engineering research hypothesis, not a claim that AI systems literally possess human psychological disorders.

### Structural Intervention

When less invasive interventions are insufficient, the system may directly modify derived semantic structures.

This is the conceptual equivalent of surgery.

Because it is more invasive, it demands stronger validation.

---

## 8.3 Quality Gates

An intervention is not considered successful merely because the agent changed.

Nothing modified by the maintenance system should automatically return to production.

The architecture specifies quality checks covering:

- schema validity
- determinism
- budget constraints
- uniqueness
- regression behavior

The prototype implements some of these gates for production output (validating generated code artifacts), but does not yet apply them to maintenance interventions on agent state. The principle remains the same regardless of what is being validated:

> **A repaired state must pass inspection before it returns to production.**

This is especially important for semantic interventions.

Changing one relationship can affect other parts of the graph.

A repair that solves one problem while creating new semantic inconsistencies is not a successful repair.

---

## 8.4 Maintenance Rotation

Persistent environments should not necessarily disappear from production while being maintained.

Partir therefore includes a maintenance-rotation model.

Multiple agents can occupy different lifecycle states:

- production
- maintenance
- diagnosis
- treatment
- validation

One agent can continue serving production traffic while another undergoes maintenance.

The original design uses 12-hour cycles.

The objective is zero-downtime maintenance for persistent agent fleets.

---

## 8.5 Prevention

Partir does not stop when one agent or graph is repaired.

If an identified failure mode could affect other agents, a validated repair can be packaged as a preventative measure.

This creates a fleet-level learning cycle:

**detect → diagnose → treat → validate → package → prevent**

A failure discovered in one part of a persistent environment can therefore become information that prevents the same failure from propagating elsewhere.

---

# 9. Research Foundation

The research behind Partir developed around the broader question:

> **Can persistent AI systems develop measurable failure patterns that require longitudinal maintenance rather than conventional error handling?**

The research material in `docs/research/` covers areas including:

- failure patterns in persistent AI agents
- probing frontier language models for measurable behavioral patterns
- synthetic psychopathology as an experimental framing
- biological sleep as a conceptual parallel for computational maintenance
- CBT-inspired structured intervention
- statistical validation of memory changes
- neuroscience research concerning structured intervention and neural reorganization

These materials should be understood as research foundations and sources of architectural inspiration.

The project does not assume that biological mechanisms map directly onto artificial systems.

Instead, it asks whether biological maintenance mechanisms suggest useful engineering strategies for systems with persistent internal state.

---

# 10. What Is in This Repository

This repository contains the full research and implementation material behind Partir.

```text
├── README.md
├── docs/
│   ├── architecture/      ← Patent application + system diagrams
│   ├── design/            ← Technical specifications for each module
│   ├── guides/            ← Prototype setup and usage guides
│   ├── reference/         ← API documentation
│   ├── research/          ← Research and supporting papers
│   └── assets/            ← Screenshots and prototype material
└── prototype/             ← Early-stage Go implementation
```

The repository is intentionally broader than a conventional software project.

The goal is to preserve the architecture and research context rather than release only the implementation.

---

# 11. The Patent

Partir began as a patent project.

The architecture was developed, researched, documented, and submitted as a provisional patent application:

**US 63/981,044 — filed February 2026**

The full provisional patent application and USPTO receipt are included under `docs/architecture/`.

The patent is being released with the rest of the project because the work could not financially be taken further through the patent process.

Rather than allowing the work to remain inaccessible, the decision was made to publish it for anyone to examine and build upon.

By publishing the patent material and supporting architecture, the project establishes a public record of the ideas and makes the work available as prior art.

---

# 12. The Prototype

`prototype/` contains an early-stage Go implementation of one potential application of the Partir architecture: a code factory.

The prototype is a node mesh that dispatches work orders to external AI executor plugins (via HTTP) and validates the returned output through quality gates. The prototype itself does not generate code — it routes prompts to plugins (like Ollama or cloud LLMs), receives whatever they produce, runs it through validation gates, and either stores the result or signals failure.

What's actually implemented:

- **Dispatch** — accepts tickets (work orders), routes them to plugins based on configuration
- **Quality gates** — Omega runs pre-flight and post-execution checks (schema validation, budget limits, determinism, uniqueness, agent output structure)
- **Defect classification** — gate failures produce typed defects (schema_violation, budget_exceeded, nondeterministic_output, etc.) stored in the database
- **Retry signaling** — the system decides whether a failure is retryable based on defect class history (same class fails twice → stop), but the dispatcher does not actually re-execute in a loop. It returns the retry recommendation to the caller.
- **Artifact storage** — validated output is stored in MinIO with provenance metadata

What's **not** implemented:

- The dispatcher doesn't retry on its own — it signals `ShouldRetry` but doesn't loop
- No maintenance, diagnostics, or drift-management from the Partir framework
- Factory worker scaling is partially wired but not functional
- Many CLI commands are stubbed or incomplete

The prototype should be understood as the dispatch and validation layer of a potential code factory, not as an implementation of the Partir framework itself.

See `prototype/INSTALL.md` for setup details.

---

# 13. What Partir Is Not

Partir is not:

- a replacement for an underlying AI model
- a general-purpose agent framework
- a claim that current AI systems are conscious
- a claim that AI systems literally experience human psychological disorders
- a turnkey production platform
- a claim that every AI failure can be solved by the interventions described here
- an empirical claim that every hypothesis in the research material has been proven

Partir is an architectural and research exploration of a specific problem:

> **How do we maintain AI environments that are expected to persist?**

---

# 14. Design Philosophy

The core philosophy can be summarized simply:

> **Persistence creates maintenance requirements.**

If an agent exists for a few minutes, restarting it may be acceptable.

If an environment is expected to operate for months or years, restarting an individual agent does not necessarily solve the problem.

The accumulated state, derived knowledge, model assumptions, handoff history, tool state, organizational requirements, and environmental context remain.

Long-lived AI environments therefore require mechanisms that can inspect, diagnose, modify, validate, and maintain that state without simply destroying it.

Partir proposes one architecture for doing that.

Its central concern is not merely whether an AI system can produce a correct answer.

It is whether the **persistent environment can remain reliable as its internal state and external environment change over time**.

That requires understanding:

**what is wrong,**

**why it is wrong,**

**where the failure originated,**

**how much the error matters,**

**what else depends on it,**

**whether the maintenance process itself introduced drift,**

and

**whether the repair actually restored system integrity.**

---

# 15. Who This Is For

Partir is intended for:

- **Researchers** studying persistent AI environments, memory, semantic drift, systemic drift, or agent reliability
- **Engineers** building long-running AI systems that need diagnostics and maintenance
- **AI infrastructure developers** working on observability, memory integrity, or agent lifecycle management
- **Researchers exploring model maintenance** and inherited model behavior
- **Anyone** investigating how AI systems can remain stable over weeks, months, or years of continuous operation

---

# 16. License

## Code

The code under `prototype/` is released under the **Apache License 2.0**.

The license includes an explicit patent grant.

## Documentation and Research

Documentation and research under `docs/` are released under **Creative Commons Attribution 4.0**.

Use, share, and adapt the material, with attribution to the project and its author.

If you build on this work, a reference to this repository and the original provisional patent filing is appreciated.

---

*Built and released by John Wagner.*
