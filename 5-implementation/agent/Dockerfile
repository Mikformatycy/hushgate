FROM python:3.12-slim
RUN pip install --no-cache-dir anthropic==0.* openai==1.*
WORKDIR /app
COPY agent.py agent_openai.py .
CMD ["sleep", "infinity"]
