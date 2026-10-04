# Claude Code in the sandbox: its only network route is the HushGate gateway.
FROM node:22-bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git ripgrep procps python3 \
    && rm -rf /var/lib/apt/lists/*
RUN npm install -g @anthropic-ai/claude-code && npm cache clean --force
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
# Owned by node so the session volume mounted here is writable.
RUN mkdir -p /home/node/.claude && chown node:node /home/node/.claude
USER node
WORKDIR /workspace
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["sleep", "infinity"]
