I’m going to define:

Core modules: Alpha, Beta, Omega, Foundry

Core artifacts + versioning model

Plugin contract + capability model

Spec-Based Ticket schema (dispatch envelope)

The minimal APIs/CLIs to ship Core v0.1

Partir – Core
Design principles

Artifacts, not vibes: every stage produces versioned artifacts.

Deterministic where possible: AI proposes; Omega decides.

Composable by plugins: Core never “knows” domains (items, NPCs, infra, etc.).

Small work units: tickets are bounded; QC is cheap.

Green Tag world: execution assumes a pinned toolchain/environment.

Intent > Alpha > Beta > Omega (Gate) > Foundry > Omega (Gate) > Artifacts

1) Core modules
1.1 Alpha Module

Purpose: define law (schemas, catalogs, rule tables, constraints).

Responsibilities

Rulebook authoring workspace (init, edit, validate)

Rulebook compilation (YAML/JSON → canonical JSON)

Versioning + promotion (immutable releases)

Serving rulebooks to Foundry and plugins

Inputs

Human edits

AI-assisted drafts (optional) as drafts only

Prior versions (for diff/review)

Outputs (Alpha Rulebook Bundle)

alpha_manifest.json (id, version, hash, created_at, parent_version)

schemas/… (JSON schema artifacts)

catalogs/… (component catalogs)

rules/… (combination rules, cost tables, invariants)

compat.json (compatibility constraints & required minimum core version)

Core guarantees

Bundle is immutable once promoted

Bundle has a stable hash for reproducibility

1.2 Beta Module

Purpose: define expression (visualization/style rulebooks, UX constraints, render vocab).

Responsibilities

Beta rulebook authoring + validation + promotion (same pattern as Alpha)

Defines visual/render grammars (e.g., glow styles, animation envelopes, palette bounds)

Can be used beyond visuals (UX flows, UI tokens) later, but Core starts simple.

Outputs (Beta Rulebook Bundle)

beta_manifest.json

render_vocab.json (effects: pulse/wave/flame/smoke/shadow etc.)

style_rules.json (bounds + propagation rules)

schemas/…

1.3 Omega Module

Purpose: authority — validates artifacts, enforces gates, classifies failures, triggers retries.

Responsibilities

Run core gates (schema validity, versions, uniqueness, reproducibility)

Run plugin gates (domain-specific validators)

Emit Defect Reports (machine-readable)

Optionally apply repairs (clamp/normalize) when permitted by policy

Inputs

Work order (ticket)

Candidate artifacts produced by plugin/executor

Referenced Alpha/Beta rulebooks

Registry state (catalog history for uniqueness, etc.)

Outputs

omega_result.json:

PASS / FAIL

defects[]

metrics (time, attempts, checks run)

defect_report.json on fail:

defect_id, class, message, offending_fields, suggested_fix

Core gate categories (built-in)

Version/compat: core version, rulebook versions, hash match

Schema: artifact schema compliance

Allowed combos: (if plugin provides a rule interpreter)

Budget/power envelope: (plugin gate)

Uniqueness: within batch + within catalog registry

Determinism: same inputs → same outputs hash (when required)

1.4 Foundry Module

Purpose: the production system — routes tickets, runs jobs, stores artifacts, commits results.

Responsibilities

Accept Spec-Based Tickets (work orders)

Load referenced rulebooks (Alpha/Beta)

Select plugin + executor

Orchestrate retries based on Omega defect reports

Store artifacts in a registry and/or repo

Emit audit logs and metrics

Foundry is not AI.

Foundry is scheduling + orchestration.

Foundry “lanes” (pluggable worker pools)

synthesis (AI proposals)

build (code generation / file edits)

render (PNG/PDF creation)

export (CSV/VTT/etc.)

assembly (combine outputs)

verify (Omega checks)

Core ships with a single worker; scaling lanes is an extension.

2) Core artifact model
2.1 Registries

Core uses registries, not databases first.

Rulebook Registry

alpha/<id>/<version>/...

beta/<id>/<version>/...

Artifact Registry

artifacts/<ticket_id>/...

catalog/<domain>/... (for uniqueness checks)

Registries can be filesystem or S3; Core abstracts storage.

2.2 Artifact typing

Every artifact has:

artifact_type (e.g., ItemSpec, RenderSpec, PDFCard)

schema_ref (optional)

hash

provenance (ticket_id, plugin_id, inputs hash, rulebook refs)

This is what lets you do DMAIC later.

3) Plugin system
3.1 Plugin concept

A plugin declares:

job types it supports

input/output schemas

which gates it requires

optional interpreters (e.g., rule evaluator)

materializers (renderers/exporters)

3.2 Minimal plugin interface

A plugin must implement:

Identity

plugin_id

version

capabilities (job types)

Schemas

input_schema(job_type)

output_schema(job_type)

Execution

plan(work_order) -> steps[] (optional)

execute(work_order, context) -> artifacts[]

Gates

gates(job_type) -> gate_ids[]

validate(artifacts, context) -> defects[] (plugin-side Omega hooks)

3.3 Executors (optional layer)

Foundry can dispatch plugin jobs to an executor:

local (Go code)

warp (shell runner)

factory_ai (agent)

cursor (agent)

human

Core defines executor contracts but ships with local only.

4) Spec-Based Ticket schema

This is the “kanban card + work order + QC checklist” envelope.

4.1 Ticket fields (core)

ticket_id (uuid)

title

plugin_id

job_type

priority (optional)

alpha_ref (id@version)

beta_ref (optional)

green_tag_ref (optional but recommended)

inputs (plugin-defined JSON)

constraints (core + plugin-defined)

acceptance_gates (list)

limits:

max_files_changed

max_diff_lines

max_attempts

max_cost_budget (tokens/$)

outputs_expected (artifact types)

routing_hints:

preferred_executor

allowed_executors

4.2 Ticket lifecycle states (Core)

SPEC_READY

DISPATCHED

IN_EXECUTION

LOCAL_QC_FAILED

READY_FOR_OMEGA

OMEGA_FAILED

COMPLETED

LINE_STOP_ESCALATED

Core emits these as events; UI integrations (Linear/Jira) come later.

5) Minimal Core APIs / CLIs
5.1 Alpha CLI/API

alpha init <id>

alpha lint <id>

alpha build <id>

alpha promote <id> --version x.y.z

alpha list

alpha get <id>@<version>

5.2 Beta CLI/API

Same as Alpha.

5.3 Foundry CLI/API

foundry submit <ticket.json>

foundry run <ticket_id>

foundry status <ticket_id>

foundry artifacts <ticket_id>

foundry export <ticket_id> --format zip

5.4 Omega (internal service or lib)

omega run --ticket <id> --artifacts <path>

returns PASS/FAIL + defect report

6) What this enables next

Once Core exists, everything else becomes plugins or higher-order products:

Game asset generators (items/NPCs/quests) → plugins

Route A 2D renderer → materializer plugin

Dual memory nodes → plugin + rulebooks

4-stage maintenance cycling → Foundry lane orchestration + Omega gates

Linear/Jira integration → ticket adapter plugin

Tiered product versions:

Core OSS (Alpha/Beta/Omega/Foundry + plugin SDK)

Pro (hosted registries, advanced Omega gates, analytics/DMAIC dashboards)

Enterprise (SSO, policy engine, audit, multi-tenant, on-prem)

7) The “v0.1 Core” milestone

To keep you from overbuilding:

Ship v0.1 with:

Rulebook registries (filesystem)

Alpha/Beta init/lint/build/promote

Foundry submit/run

Omega core gates (schema/version/uniqueness/determinism)

Plugin SDK (in-process)

One sample plugin: game_items (can output JSON only)

One materializer plugin: render_2d (optional)