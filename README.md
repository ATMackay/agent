# Agent CLI - AI Agents with Google ADK

This is a toy project to build AI agents using a pure Go stack and explore the capabilities of Google's [ADK](https://google.github.io/adk-docs/get-started/go/).

## Getting started 

Run the documentation agent on this project


Export API key (Gemini, Claude)
```
export API_KEY=AI...Zs
```

Build agent CLI
```
make build
```

Run the agent ()
```
./build/agent-cli run documentor --repo https://github.com/ATMackay/agent.git --output agentcli-doc.md
```