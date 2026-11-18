# AI Inference Labs — Proposed Editorial Calendar

**Window:** November 10, 2026 – January 29, 2027 (12 weeks)  
**Audience:** AI infrastructure engineers, SREs and platform engineers  
**Channels:** Long-form on Medium, reproducible code and diagrams on GitHub, short-form distribution on LinkedIn/Medium  
**Status:** Proposal — not a publishing commitment  
**Output target:** 12 research/architecture explainers + 12 hands-on tutorials + 12 short posts

## The editorial idea

Explain difficult AI infrastructure problems as engineers actually encounter them. Each week's research article explains an idea and its tradeoffs; the tutorial builds an observable piece of that idea; the short post highlights one practical lesson. The Go-based gateway grows across tutorials rather than spawning twelve unrelated applications.

Direct, human prose; no invented experience, measurements, quotes, or paper results. The first gateway tutorial starts with *why a gateway is useful*, compares alternatives, explains streaming and SSE, walks through the source, and then shows how to build and run it locally.

## Weekly publication schedule

| Week | Tue: Research / technical discussion | Topic | Thu: Step-by-step tutorial | Topic | Fri: Short post | Topic |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 2026-11-10 | Why LLM Inference Is Two Different Problems: Prefill and Decode | 2026-11-12 | Building a Streaming AI Inference Gateway in Go | 2026-11-13 | Why TTFT and tokens per second tell different stories |
| 2 | 2026-11-17 | KV Cache Is Changing How We Design AI Infrastructure | 2026-11-19 | Build a Load-Aware Inference Router in Go | 2026-11-20 | Why round-robin fails for some inference workloads |
| 3 | 2026-11-24 | A Scheduler Can Be Fast and Still Make Bad Decisions | 2026-11-26 | Add Request Queues, Backpressure, and Admission Control | 2026-11-27 | Utilization and goodput are not the same metric |
| 4 | 2026-12-01 | What Traditional APM Misses in AI Inference | 2026-12-03 | Instrument Your Inference Router with OpenTelemetry | 2026-12-04 | Three inference metrics worth tracking first |
| 5 | 2026-12-08 | Should Prefill and Decode Run on Separate GPUs? | 2026-12-10 | Simulate Prefill/Decode Disaggregation in Go | 2026-12-11 | When KV-cache transfer costs more than it saves |
| 6 | 2026-12-15 | Can a Router Use KV Cache Locality to Reduce Latency? | 2026-12-17 | Add Prefix-Cache-Aware Routing with a Mock Backend | 2026-12-18 | Cache hits change placement decisions |
| 7 | 2026-12-22 | Why Low GPU Utilization Does Not Necessarily Mean Low Cost | 2026-12-24 | Build an Inference Cost and Capacity Calculator | 2026-12-25 | Cost per useful token versus cost per GPU-hour |
| 8 | 2026-12-29 | Can KV Cache Cross a Datacenter Boundary? | 2026-12-31 | Build a Latency-Aware Multi-Region Router | 2027-01-01 | The nearest region isn't always the best choice |
| 9 | 2027-01-05 | What Happens When an Inference Cluster Disappears? | 2027-01-07 | Add Circuit Breakers, Failover, and Request Budgets | 2027-01-08 | Why retry storms make GPU clusters less reliable |
| 10 | 2027-01-12 | AI Inference Scheduling Is Becoming a Power Problem | 2027-01-14 | Build a Capacity-Aware Scheduler Simulation | 2027-01-15 | When queueing is better than adding another GPU |
| 11 | 2027-01-19 | Kubernetes Isn't an AI Scheduler—and That's Fine | 2027-01-21 | Deploy the Go Router to a Local Kubernetes Cluster | 2027-01-22 | What Kubernetes should and shouldn't schedule |
| 12 | 2027-01-26 | What Would a Production Inference Control Plane Look Like? | 2027-01-28 | Harden the Gateway: CI, Load Tests, and Deployment | 2027-01-29 | What I'd change before putting this in production |

## Proposed weekly work breakdown

1. **Research:** Find and read original papers and primary implementation docs. Verify claims, dates and benchmark conditions; distinguish paper results from general production performance.
2. **Build:** Extend one Go project. Keep examples runnable locally using Go, curl, a mock backend and optional Docker/kind. No paid GPU required for the baseline labs.
3. **Illustrate:** Create **one cover thumbnail plus at least one explanatory infographic or architecture diagram** for every long-form article. Diagrams must match the actual implementation. Prefer editable SVG for technical diagrams; maintain attribution for externally sourced images.
4. **Write:** Explain why the problem matters before introducing the implementation. For tutorials: alternatives, protocol background, top-down code walkthrough, build/run/curl demonstration, failure modes, optional real model extension.
5. **Review:** Validate code, source links, code-fence accuracy, screenshots/diagrams and release/commit pin. Publish only after review; automated tests remain part of CI, not the main reader experience.
6. **Distribute:** Publish the Medium article, link the matching GitHub lab, and post a shorter takeaway with the canonical article URL.

## Research candidates to verify when drafting

These are **starting points**, not pre-approved claims or a fixed reading list:

- [PagedAttention / vLLM](https://arxiv.org/abs/2309.06180) — KV-cache memory management.
- [DistServe](https://arxiv.org/abs/2401.09670) — splitting prefill and decode.
- [SGLang](https://arxiv.org/abs/2312.07104) — structured serving and reuse.
- [NVIDIA Dynamo documentation](https://docs.nvidia.com/dynamo/latest/) — distributed inference architecture.
- [vLLM documentation](https://docs.vllm.ai/) — serving and metrics.
- Relevant new papers should be evaluated close to their assigned publication week rather than chosen based only on novelty.

## GitHub code roadmap

| Week | Planned lab or milestone | Minimum concrete deliverable |
| --- | --- | --- |
| 1 | `01-streaming-gateway` | Buildable streaming HTTP gateway and simulated SSE backend |
| 2 | `02-load-aware-routing` | Multiple backends and a measurable selection policy |
| 3 | `03-admission-control` | Bounded queues, admission policy and rejection examples |
| 4 | `04-observability` | Tracing and request/stream latency metrics |
| 5 | `05-disaggregation` | Separate simulated prefill/decode worker behaviors |
| 6 | `06-cache-aware-routing` | Cache-locality metadata and configurable routing policy |
| 7 | `07-capacity-cost` | Local workload and cost estimator with stated assumptions |
| 8 | `08-multi-region` | Regional backend simulation with latency-aware selection |
| 9 | `09-resilience` | Failure handling, cancellation and circuit breakers |
| 10 | `10-capacity-scheduler` | Capacity planning simulation with workload fixtures |
| 11 | `11-kubernetes` | Local kind deployment and manifest documentation |
| 12 | `12-production` | Release/CI, reproducible load test and production checklist |

The names above describe milestones, not existing directories or completed code. Keep the application package structure reusable and use a git tag or commit permalink for each published tutorial.

## Current drafting status

- **Week 1 research explainer:** drafted in [PR #1](https://github.com/afoley587/ai-inference-labs/pull/1).
- **Week 1 streaming gateway tutorial:** drafted and revised in [PR #1](https://github.com/afoley587/ai-inference-labs/pull/1).
- **Week 1 short post:** drafted in [PR #1](https://github.com/afoley587/ai-inference-labs/pull/1).
- No publication to Medium is represented by this schedule. PR #1 is separate from this calendar branch.

## Notes on timing

- Week 3 overlaps US Thanksgiving (November 26); the November 26 tutorial is a target, not a promise.
- Weeks 7–8 overlap December 24–25 and December 31–January 1; plan to pre-write, reschedule or skip holiday releases based on capacity.
- The plan assumes a repeatable Tuesday research explainer, Thursday tutorial, Friday short post cadence. A two-week draft buffer is recommended before public publication.
- Dates are proposals, not scheduled Medium publications; publishing requires review and an authenticated publishing workflow.

## How to maintain the schedule

The [editable CSV](editorial-calendar.csv) stores one row per individual publication, including status, proposed date, title, repo milestone and format. Use `Planned → Research → Draft → Technical review → Approved → Published`; add the published Medium URL and GitHub commit permalink after release. If a topic becomes stale or a paper's results don't hold up, change the title rather than forcing the schedule.
