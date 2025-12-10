FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir requests networkx

COPY analyze.py .

CMD ["python", "analyze.py"]