FROM python:3.13-slim

# Set working directory
WORKDIR /app

# Install system dependencies including certificates
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Copy requirements and install Python dependencies
COPY requirements.txt .
RUN pip install --trusted-host pypi.org --trusted-host pypi.python.org --trusted-host files.pythonhosted.org --no-cache-dir -r requirements.txt

# Copy simulator code  
COPY ethernet_ip_server.py .

# Create non-root user for security
RUN useradd -m -u 1000 simulator

# Expose Ethernet/IP ports
EXPOSE 44818/udp
EXPOSE 2222

USER simulator

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD python -c "import socket; s=socket.socket(socket.AF_INET, socket.SOCK_DGRAM); s.settimeout(1); s.bind(('', 0)); s.close()" || exit 1

# Default command
CMD ["python", "ethernet_ip_server.py", "--log-level", "INFO"]