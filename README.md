# AI Inference Labs

Readable engineering articles paired with small, reproducible experiments about AI inference systems. Written for infrastructure engineers working with networks, GPUs, scheduling, observability and cloud platforms.

## Start reading

| Format | Topic | Visuals |
| --- | --- | --- |
| Research explainer | [Why LLM Inference Is Two Different Problems: Prefill and Decode](articles/01-prefill-vs-decode.md) | [Cover](assets/01-prefill-decode-cover.svg) · [Flow](assets/01-prefill-decode-flow.svg) · [Latency](assets/01-latency-timeline.svg) |
| Go tutorial | [Building a Streaming AI Inference Gateway in Go](articles/02-streaming-gateway.md) | [Cover](assets/02-streaming-gateway-cover.svg) · [Stream diagram](assets/02-streaming-flow.svg) |
| Short post | [Why round-robin isn't always enough for inference](posts/01-prefill-decode-field-note.md) | — |

## Run lab 01: streaming gateway

Requires Go 1.22+ and no external dependencies.

```bash
go test ./...
go run ./cmd/mock-backend
```

In another terminal:

```bash
BACKEND_URL=http://localhost:8081 go run ./cmd/gateway
```

And in a third terminal:

```bash
curl -N http://localhost:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"Hello"}'
```

The backend sends **synthetic** SSE events. It does not host a language model, and timings are not real inference measurements.

## Roadmap

1. Streaming gateway and inference fundamentals — draft
2. Multiple backends and load-aware routing
3. Admission control, queueing and backpressure
4. OpenTelemetry and serving metrics
5. Prefill/decode separation simulation
6. Cache-aware routing
7. Cost, capacity and multi-region placement
8. Failure handling and Kubernetes deployment

## Editorial principles

- Short, useful explanations grounded in real engineering constraints.
- Primary citations for research claims.
- Code and validation instructions alongside every tutorial.
- SVG diagrams and thumbnail per long-form article.
- Simulated examples clearly separated from real measurements.
- No invented incidents, experiments or performance results.

## Status and license

The initial labs are in development. CI should be reviewed before merging. A repository-wide license has not been selected, so don't assume permission to reuse the source beyond the rights granted by GitHub's terms.
