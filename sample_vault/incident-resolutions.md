# Incident Resolutions & Known Fixes

Tags: #incidents #runbook #troubleshooting

## Incident: Crucible Container Timeout (Exit Code 124 / SIGKILL)
- **Symptoms**: AI execution tests exceed 5 seconds in [[foundry-crucible]], resulting in timeout error payload: `{"error": "execution timed out after 5.0 seconds"}`.
- **Root Cause**: Infinite loops, unbuffered I/O, or network calls blocked by `network_mode=none`.
- **Known Fix**:
  1. Inspect script loop bounds.
  2. Strip external socket attempts or package downloads; run pure local calculations.
  3. Ensure code terminates cleanly within 2 seconds.
- **Reference**: See [[agent-playbooks]].

## Incident: OOM Kill (Exit Code 137)
- **Symptoms**: Container exits abruptly with code 137 in [[foundry-crucible]].
- **Root Cause**: Python process allocated > 128MB RAM during matrix creation or memory leak.
- **Known Fix**: Use streaming iterators or generators instead of loading full arrays into memory.
