# Virtual Device Testing Framework Makefile

# Color codes for output
RED = \033[0;31m
GREEN = \033[0;32m
BLUE = \033[0;34m
YELLOW = \033[1;33m
NC = \033[0m # No Color

.PHONY: help build up down logs test clean status restart

help: ## Show this help message
	@echo "$(BLUE)🌉 Virtual Device Testing Framework$(NC)"
	@echo "======================================"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

build: ## Build all virtual device containers
	@echo "$(BLUE)🔨 Building virtual device containers...$(NC)"
	@docker compose build

up: ## Start basic virtual device environment (core protocols)
	@echo "$(BLUE)🚀 Starting basic virtual device environment...$(NC)"
	@docker compose up -d modbus-tcp-sim modbus-rtu-sim opcua-sim s7-sim ethernet-ip-sim
	@echo "$(GREEN)✅ Basic environment started$(NC)"
	@echo "  - Modbus TCP: localhost:502"
	@echo "  - Modbus RTU: localhost:503"
	@echo "  - OPC UA: opc.tcp://localhost:4840"
	@echo "  - S7 PLC: localhost:102"
	@echo "  - Ethernet/IP: localhost:44818/udp, localhost:2222"

up-extended: ## Start extended environment with additional device variants
	@echo "$(BLUE)🚀 Starting extended virtual device environment...$(NC)"
	@docker compose --profile extended up -d
	@echo "$(GREEN)✅ Extended environment started$(NC)"

up-testing: ## Start testing environment with error simulation
	@echo "$(BLUE)🧪 Starting testing environment with error simulation...$(NC)"
	@docker compose --profile testing up -d
	@echo "$(GREEN)✅ Testing environment started$(NC)"

up-all: ## Start all virtual devices
	@echo "$(BLUE)🚀 Starting all virtual devices...$(NC)"
	@docker compose --profile extended --profile testing up -d
	@echo "$(GREEN)✅ All virtual devices started$(NC)"

down: ## Stop and remove all virtual device containers
	@echo "$(BLUE)🛑 Stopping virtual device environment...$(NC)"
	@docker compose --profile extended --profile testing down
	@echo "$(GREEN)✅ Environment stopped$(NC)"

restart: down up ## Restart the basic environment

logs: ## View logs from all running containers
	@echo "$(BLUE)📋 Viewing container logs...$(NC)"
	@docker compose logs -f

logs-tail: ## View last 50 lines of logs from all containers
	@echo "$(BLUE)📋 Viewing recent container logs...$(NC)"
	@docker compose logs --tail=50

status: ## Show status of all containers
	@echo "$(BLUE)📊 Container Status:$(NC)"
	@echo "=================="
	@docker compose ps
	@echo ""
	@echo "$(BLUE)🌐 Network Status:$(NC)"
	@echo "=================="
	@docker network ls | grep virtual-devices || echo "No virtual-devices network found"

test: ## Run virtual device test suite
	@echo "$(BLUE)🧪 Running virtual device tests...$(NC)"
	@python3 test_virtual_devices.py --timeout 15

test-quick: ## Run quick connectivity tests
	@echo "$(BLUE)⚡ Running quick connectivity tests...$(NC)"
	@python3 test_virtual_devices.py --timeout 5

clean: ## Clean up containers, images, and networks
	@echo "$(BLUE)🧹 Cleaning up virtual device resources...$(NC)"
	@docker compose --profile extended --profile testing down --volumes --remove-orphans
	@echo "$(YELLOW)🗑️  Removing virtual device images...$(NC)"
	@docker images | grep virtual-devices | awk '{print $$3}' | xargs -r docker rmi || echo "No images to remove"
	@echo "$(GREEN)✅ Cleanup complete$(NC)"

rebuild: clean build ## Clean and rebuild all containers

# Individual device commands
modbus-tcp: ## Start only Modbus TCP simulator
	@echo "$(BLUE)🔌 Starting Modbus TCP simulator...$(NC)"
	@docker compose up -d modbus-tcp-sim
	@echo "$(GREEN)✅ Modbus TCP simulator available at localhost:502$(NC)"

modbus-rtu: ## Start only Modbus RTU simulator
	@echo "$(BLUE)🔌 Starting Modbus RTU simulator...$(NC)"
	@docker compose up -d modbus-rtu-sim
	@echo "$(GREEN)✅ Modbus RTU simulator available at localhost:503$(NC)"

opcua: ## Start only OPC UA simulator
	@echo "$(BLUE)🔌 Starting OPC UA simulator...$(NC)"
	@docker compose up -d opcua-sim
	@echo "$(GREEN)✅ OPC UA simulator available at opc.tcp://localhost:4840$(NC)"

s7: ## Start only S7 PLC simulator
	@echo "$(BLUE)🔌 Starting S7 PLC simulator...$(NC)"
	@docker compose up -d s7-sim
	@echo "$(GREEN)✅ S7 PLC simulator available at localhost:102$(NC)"

ethernet-ip: ## Start only Ethernet/IP simulator
	@echo "$(BLUE)🔌 Starting Ethernet/IP simulator...$(NC)"
	@docker compose up -d ethernet-ip-sim
	@echo "$(GREEN)✅ Ethernet/IP simulator available at localhost:44818/udp$(NC)"

# Development and debugging commands
debug-modbus: ## Debug Modbus TCP simulator with verbose logging
	@echo "$(BLUE)🐛 Starting Modbus TCP in debug mode...$(NC)"
	@docker compose run --rm -p 502:502 modbus-tcp-sim python modbus_server.py --log-level DEBUG

debug-opcua: ## Debug OPC UA simulator with verbose logging
	@echo "$(BLUE)🐛 Starting OPC UA in debug mode...$(NC)"
	@docker compose run --rm -p 4840:4840 opcua-sim python opcua_server.py --log-level DEBUG

shell-modbus: ## Open shell in Modbus TCP container
	@docker compose exec modbus-tcp-sim /bin/bash || echo "Container not running. Use 'make up' first."

shell-opcua: ## Open shell in OPC UA container
	@docker compose exec opcua-sim /bin/bash || echo "Container not running. Use 'make up' first."

# Documentation and information
protocols: ## Show supported protocols and their ports
	@echo "$(BLUE)📡 Supported Industrial Protocols:$(NC)"
	@echo "=================================="
	@echo "$(GREEN)Modbus TCP$(NC)        - Port 502  (Standard)"
	@echo "$(GREEN)Modbus RTU$(NC)        - Port 503  (Energy Meter)"
	@echo "                     Port 504  (Temperature Controller)"
	@echo "$(GREEN)OPC UA$(NC)            - Port 4840 (Factory Server)"
	@echo "                     Port 4841 (Process Server)"
	@echo "$(GREEN)S7 (Siemens)$(NC)      - Port 102  (PLC Communication)"
	@echo "$(GREEN)Ethernet/IP$(NC)       - Port 44818/udp (Discovery)"
	@echo "                     Port 2222 (Explicit Messaging)"
	@echo ""
	@echo "$(BLUE)🧪 Testing Variants:$(NC)"
	@echo "==================="
	@echo "$(YELLOW)Faulty Modbus$(NC)     - Port 505  (Error Simulation)"

network-info: ## Show network configuration
	@echo "$(BLUE)🌐 Network Configuration:$(NC)"
	@echo "========================="
	@echo "$(GREEN)Network Name:$(NC) virtual-devices_industrial-net"
	@echo "$(GREEN)Subnet:$(NC) 172.20.0.0/16"
	@echo "$(GREEN)Driver:$(NC) bridge"
	@echo ""
	@echo "$(BLUE)Container IPs:$(NC)"
	@docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "No containers running"