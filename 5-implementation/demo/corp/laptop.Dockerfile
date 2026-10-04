FROM python:3.12-slim
RUN pip install --no-cache-dir anthropic openai httpx
WORKDIR /app
COPY agent/agent.py agent/agent_openai.py demo/corp/laptop.py demo/corp/laptop-entrypoint.sh ./
ENTRYPOINT ["/app/laptop-entrypoint.sh"]
CMD ["sleep", "infinity"]
