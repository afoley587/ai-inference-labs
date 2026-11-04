# Why round-robin isn't always enough for inference

Two GPUs can have similar utilization and very different amounts of useful capacity.

One might be processing a large prompt. Another might be decoding many concurrent requests, with little KV-cache memory left for the next sequence.

A generic load balancer sees two healthy endpoints. An inference-aware scheduler needs to ask a different question: which backend can admit this particular request without damaging latency for work already in progress?

That's why separating prefill and decode matters. They share a request, but not the same performance profile.

The first article in AI Inference Labs explains the tradeoffs and where to start measuring.

Read: [Why LLM Inference Is Two Different Problems](../articles/01-prefill-vs-decode.md)
