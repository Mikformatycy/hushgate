FROM python:3.12-slim
WORKDIR /app
COPY demo/fake_upstream.py demo/corp/internet.py ./
CMD ["python", "-u", "internet.py"]
