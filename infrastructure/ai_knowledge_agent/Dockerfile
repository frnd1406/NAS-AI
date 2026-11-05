# Multi-stage build for smaller image size (Alpine-based for security)
FROM python:3.13-alpine AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# System deps for building psycopg2 on Alpine
RUN apk add --no-cache \
    build-base \
    postgresql-dev \
    musl-dev

COPY requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

# Final runtime stage
FROM python:3.13-alpine

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

WORKDIR /app

# Only install runtime dependencies
# SECURITY: Alpine has faster security updates than Debian
RUN apk add --no-cache \
    libpq

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.13/site-packages /usr/local/lib/python3.13/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

# pip (and the build-time tooling vendored in it) is not needed at runtime
RUN pip uninstall -y pip setuptools wheel 2>/dev/null; rm -rf /usr/local/lib/python3.13/site-packages/pip* /usr/local/bin/pip*

COPY src /app/src

EXPOSE 5000

# Run as non-root user for security
RUN adduser -D -u 1000 appuser && chown -R appuser:appuser /app
USER appuser

CMD ["gunicorn", "-w", "2", "-b", "0.0.0.0:5000", "--timeout", "120", "--access-logfile", "-", "src.main:app"]
