# Agent Playbooks & Remediation Workflows

Tags: #playbook #agents #automation

## Autonomous Remediation Loop in [[foundry-omni]]
When an incident is reported by [[foundry-pulse]]:
1. **Retrieve Context**: Query [[foundry-graph]] for past incidents and architectural invariants.
2. **Draft Patch**: Use [[foundry-router]] to generate minimal diffs adhering to [[architecture-decisions]].
3. **Sandbox Verification**: Test the fix in [[foundry-crucible]] against unit tests.
4. **Airgap Commit**: If tests return status 200 and exit code 0, stage and commit the change via [[foundry-airgap]].
