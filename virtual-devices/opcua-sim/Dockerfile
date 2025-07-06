FROM python:3.13-slim

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy simulator code
COPY opcua_server.py .

# Make the script executable
RUN chmod +x opcua_server.py

# Expose OPC UA port
EXPOSE 4840

# Create non-root user for security
RUN useradd -m -u 1000 simulator
USER simulator

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import socket; s=socket.socket(); s.settimeout(1); s.connect(('localhost', 4840)); s.close()" || exit 1

# Default command
CMD ["python", "opcua_server.py", "--log-level", "INFO"]