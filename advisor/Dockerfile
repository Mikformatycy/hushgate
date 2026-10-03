FROM python:3.12-slim
RUN pip install --no-cache-dir httpx
WORKDIR /app
COPY advisor.py .
CMD ["python", "-u", "advisor.py"]
