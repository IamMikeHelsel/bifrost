# OPC UA Simulators

## Purpose
Full OPC UA server/client simulators for comprehensive OPC UA testing.

## Planned Contents
- **Server Simulator**: Complete OPC UA server with configurable namespace
- **Client Simulator**: OPC UA client behavior simulation
- **Security Simulator**: All security policies and certificate scenarios
- **Historical Access**: Historical data simulation and retrieval

## Key Features
- Complete OPC UA server implementation
- All security policies (None, Basic128Rsa15, Basic256, Basic256Sha256)
- Subscription and monitored item support
- Method call simulation
- Event notification system
- Historical data access

## Performance Targets
- 10,000+ tags/second read throughput
- 1,000+ concurrent subscriptions
- < 10ms subscription update latency
- Browse 10,000 nodes in < 1 second

## Testing Focus
- Security policy validation
- Subscription performance
- Large namespace browsing
- Certificate management
- High-frequency data collection