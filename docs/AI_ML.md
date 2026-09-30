# AI & Machine Learning Specification

## 1. Architectural Boundaries

Artificial Intelligence (AI) and Machine Learning (ML) subsystems within SentinelExam are strictly **advisory** and **read-only**.

### Invariants:
1. **Never Authoritative**: AI models cannot mutate attempt state, auto-fail candidates, or modify candidate scores directly.
2. **Model-Agnostic Abstraction**: Integrated through a unified LiteLLM proxy or model-agnostic gateway.
3. **Graceful Degradation**: Outages or latency spikes in AI providers MUST NOT impede or block the candidate assessment runtime.
4. **Health Probe Independence**: AI provider failure will report `ai_service: DEGRADED` under `/health/ready` but will NOT make the readiness probe return `503`.

## 2. Use Cases

* **Proctor Assistance**: Highlighting suspicious gaze shifts, audio anomalies, or rapid keystroke bursts for human proctor review.
* **Rubric & Scoring Suggestions**: Providing suggested evaluations on free-response questions for human grader approval.
* **Item Quality Analysis**: Flagging ambiguous question stems or high discriminatory error rates in QuestionBanks.
