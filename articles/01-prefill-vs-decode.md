---
title: "Why LLM Inference Is Two Different Problems: Prefill and Decode"
subtitle: "How prefill and decode put different demands on your infrastructure, and what that means for scheduling."
status: draft
series: AI Inference Labs
tags: [llm-inference, gpu, scheduling, kv-cache, infrastructure]
thumbnail: ../assets/01-prefill-decode-cover.svg
---

![Prefill and decode: two phases, two infrastructure problems](../assets/01-prefill-decode-cover.svg)

# Why LLM Inference Is Two Different Problems: Prefill and Decode

A busy GPU doesn't always mean a responsive service. You can have plenty of compute capacity on paper and still leave users waiting for the first token. When it finally arrives, the rest of the answer may trickle out.

Adding another GPU might help, but it depends on what's causing the delay. We get a clearer view by looking at the two jobs bundled together under *inference*: **prefill** and **decode**.

For infrastructure engineers, this distinction affects how we investigate slow requests and decide when to add capacity. A long wait before the first token may come from queueing or expensive prompt processing. An answer that starts quickly but repeatedly pauses calls for a closer look at generation scheduling and memory pressure. Network delays or response buffering can affect either symptom, so we need measurements along the request path.

These investigations can lead to different fixes. If we track only GPU utilization and total request latency, we can spend money without improving the delay users actually notice.

## The two stages of a request

An **autoregressive** language model generates each new token using the prompt and the tokens it has already produced. A **token** is a unit of text, often a word or part of one. We'll focus here on conventional transformer-based text generation; other model architectures and generation techniques can change the execution details.

During **prefill**, the model processes the prompt tokens. In a transformer, this builds a **key-value (KV) cache**, which stores attention data the model can reuse during generation. Much of the prompt computation can run in parallel, though attention dependencies and implementation details still matter. The prefill pass also typically produces the scores, called *logits*, used to select the first output token.

During **decode**, the model processes the previously generated token, adds its attention keys and values to the cache, and selects the next token. That loop repeats until generation stops. Successive tokens within one request depend on the earlier ones, while a serving engine can batch decode work from multiple requests together.

![Request path through prefill, KV cache and autoregressive decode](../assets/01-prefill-decode-flow.svg)

The two stages may execute on the same GPU, but they place different demands on it.

| | Prefill | Decode |
| --- | --- | --- |
| Work | Process many input tokens | Produce the next output token, repeatedly |
| Parallelism | Substantial parallelism across prompt tokens | Sequential dependence between successive tokens for one request |
| Common performance pressure | Compute and prompt length | Model-weight and KV-cache memory traffic, plus batching behavior |
| User-facing symptom | Slow first token | Slow tokens after the first |
| Metric to watch | TTFT, with queue and network effects | Individual inter-token gaps and average TPOT |

Processing many prompt tokens at once gives the GPU substantial parallel computation, so prefill is often compute-bound: arithmetic throughput limits progress. A decode step processes only one new token per sequence, while still reading model weights and cached attention data. At small batch sizes, it is often memory-bandwidth-bound: moving data from GPU memory takes more time than the arithmetic itself. [NVIDIA's inference optimization overview](https://developer.nvidia.com/blog/mastering-llm-techniques-inference-optimization/) describes these different demands.

These tendencies give us a starting point. Larger decode batches can reuse weight reads across more tokens, changing the balance between computation and memory traffic. The limiting resource also depends on context length and model architecture.

## Why the first token is its own problem

Time to first token (TTFT) measures how long the client waits for the initial generated token. Prefill accounts for only part of that wait. A request also spends time in the router and the model server's queue, with network delays along the way.

When TTFT rises, we'd look at where that extra time went before tuning the GPU.

Consider a request with a long input and a short output. Most of its useful work could happen before the first visible token. A chat request with a short input and a long output may spend more of its time in decode. These are examples to help reason about latency, rather than benchmark results.

![An illustrative latency timeline showing queue, prefill and decode](../assets/01-latency-timeline.svg)

Optimizing average end-to-end latency without separating these cases can hide the problem users notice most.

After the first token, **inter-token latency** measures the gaps between successive output tokens. **Time per output token (TPOT)** commonly describes the average time per subsequent token: `(total request latency − TTFT) / (output tokens − 1)`, for responses with at least two tokens. A reasonable average can still hide a long pause halfway through an answer. Streaming chunks can also contain several tokens, so client measurements need to account for how output is delivered. Check your benchmarking tool's definitions before comparing results; [NVIDIA's metrics documentation](https://docs.nvidia.com/nim/benchmarking/llm/metrics) notes that implementations vary.

## Decode makes memory management part of scheduling

The KV cache lets the model reuse attention keys and values from earlier tokens. That saves repeated computation, while giving the serving system a growing cache to manage.

KV-cache size grows with sequence length and the model's attention dimensions. For one sequence in a conventional full-attention transformer, the approximate logical storage is:

`KV bytes ≈ 2 × layers × KV heads × head dimension × tokens × bytes per element`

The factor of two accounts for both keys and values. `Layers` is the number of attention layers; `KV heads` is the number of key-value heads per layer, and `head dimension` is the number of elements in each head's key or value vector. `Tokens` counts cached context tokens, including generated tokens already processed by the model. `Bytes per element` depends on the cache's data type.

For multiple active sequences, add their logical cache requirements, accounting for any shared prefixes. This estimate excludes model weights and other runtime memory. Grouped-query attention changes the KV-head count, while cache quantization changes bytes per element. Paging affects allocation overhead, and distributing the model across GPUs changes how much cache each device holds.

The [PagedAttention paper](https://arxiv.org/abs/2309.06180) examines how fragmentation and over-reservation waste space in variable-length KV caches, reducing serving throughput. vLLM uses paged KV-cache management to make better use of that space. The improvements reported in the paper apply to its evaluated workloads; we'd need to measure our own deployment to know what to expect.

Now imagine two identical backends. One has low GPU utilization but almost no room for another long-running sequence. The other reports higher utilization and has enough free cache capacity to admit a new request.

If we give the router only a utilization metric, it has no way to see that difference.

## Prefill and decode can interfere with each other

When prefill and decode share workers, a large prompt can compete with ongoing token generation. Scheduling policies and batching reduce or amplify that interference.

One way to manage this is **chunked prefill**. The serving engine splits a long prompt's processing into smaller pieces and interleaves them with decode work on shared workers. This gives ongoing requests opportunities to generate tokens while the new prompt is being processed. The chunk size affects the balance between first-token latency and smooth generation; [vLLM's scheduling documentation](https://docs.vllm.ai/en/v0.22.1/configuration/optimization/) describes that tradeoff.

**Disaggregated prefill/decode** takes a different architectural approach: prompt processing and token generation run on separate worker pools, typically using different GPUs. The prefill worker transfers the request's KV cache to a decode worker, which continues generation. This lets the serving system allocate capacity and tune scheduling independently for each phase. The [DistServe research](https://arxiv.org/abs/2401.09670) studies this approach with resource allocation tuned to time-to-first-token and time-per-output-token targets.

Moving the cache costs bandwidth and time, and the system now has separate worker pools to balance. We'd weigh the gains from specialization against those transfer and orchestration costs before separating the phases in a deployment. Shared workers with chunked prefill may be a better fit when those costs outweigh the benefits of disaggregation.

A paper can show improvement on its testbed while your deployment sees the opposite result. The outcome depends on your network topology and model size, as well as concurrency and prompt distribution. It also depends on your **service-level objectives (SLOs)**: the targets you set for acceptable service, such as a TTFT threshold met by a specified percentage of requests.

## What this changes for infrastructure engineers

Separating the phases helps connect a user-visible delay to an operational decision.

**Routing and admission.** A generic least-connections or round-robin policy doesn't know whether a destination has room for another large context. A backend with fewer requests may already hold long sequences that consume most of its cache capacity. Routing needs signals about queued work and available cache space; admission control needs a policy for how much additional context the worker can safely accept.

**Scheduling.** With shared workers, chunked prefill gives us a way to balance first-token waits against pauses in ongoing answers. Disaggregation lets us tune the phases independently, while adding a cache-transfer path and separate capacity pools to manage. We'd compare both against the workload's latency targets before choosing an architecture.

**Capacity planning and scaling.** The mix of prompt-heavy and output-heavy requests affects achievable **goodput**, the rate of requests completed while meeting latency objectives. An autoscaler should be evaluated against the SLO it's meant to protect. Rising queue time and TTFT suggest a different investigation from worsening decode latency under sustained concurrency. GPU utilization alone doesn't tell you which limit you've reached, or whether another replica will relieve it.

**Observability.** Measure queue time separately from client-observed TTFT, and track the distribution of inter-token gaps alongside average TPOT. Include input and output token counts so you can compare similar workloads. Backend cache occupancy and errors help explain what those latency measurements mean. Tail measurements, such as the 95th percentile, show delays an average can conceal.

We'd start by collecting those measurements. They can tell us whether a more complicated scheduler would help.

## A useful experiment

Start with two mock inference backends, each with a different simulated queue delay and token interval. Record TTFT and inter-token gaps while routing traffic round-robin. Then compare those measurements with a simple load-aware policy.

The simulation lets us see what the router can infer from the signals we give it. But we'll still need a real GPU workload to test hardware behavior.

That's the idea behind the companion [Go streaming gateway tutorial](02-streaming-gateway.md). We start by forwarding a streamed response correctly. Later labs introduce queueing and telemetry, giving us the signals to try more informed routing.

## Make the phases visible

Prefill and decode put different demands on your infrastructure, even within a single request. Measuring them separately gives you a better basis for scheduling and capacity decisions. Start there, and use what you find to decide where the next infrastructure investment should go.

## Terms at a glance

| Term | Meaning in this article |
| --- | --- |
| Token | A unit of text processed or generated by the model, often a word or part of one. |
| Autoregressive | Generating each new token conditioned on the prompt and previously generated tokens. |
| Prefill | Processing the prompt to build cached attention state and typically produce the scores for the first output token. |
| Decode | Repeatedly processing generated tokens to produce subsequent output tokens. |
| Chunked prefill | Splitting prompt processing into smaller pieces that can be scheduled alongside decode work on shared workers. |
| Disaggregated prefill/decode | Running prefill and decode on separate worker pools, with cached attention state transferred between them. |
| KV cache | Stored attention keys and values reused to avoid recomputing them for earlier tokens. |
| TTFT | Time to first token: the client's wait from submitting a request to receiving the first output token. |
| Inter-token latency | The time between successive output tokens; individual gaps reveal uneven delivery. |
| TPOT | Time per output token, commonly averaged over generation after the first token. |
| SLO | Service-level objective: a measurable target for acceptable service, often with a latency threshold and a required percentage of requests meeting it. |
| Goodput | The rate of requests completed while meeting the specified service objectives. |

### Sources and further reading

- Kwon et al., *Efficient Memory Management for Large Language Model Serving with PagedAttention*, SOSP 2023: https://arxiv.org/abs/2309.06180
- Zhong et al., *DistServe: Disaggregating Prefill and Decoding for Goodput-optimized Large Language Model Serving*, 2024: https://arxiv.org/abs/2401.09670
- NVIDIA, *Mastering LLM Techniques: Inference Optimization*: https://developer.nvidia.com/blog/mastering-llm-techniques-inference-optimization/
- NVIDIA, *LLM Inference Benchmarking: Fundamental Concepts*: https://developer.nvidia.com/blog/llm-benchmarking-fundamental-concepts/
- NVIDIA, *NIM Benchmarking Metrics*: https://docs.nvidia.com/nim/benchmarking/llm/metrics
- Companion lab: [Streaming gateway in Go](02-streaming-gateway.md)
