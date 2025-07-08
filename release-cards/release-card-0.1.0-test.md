# Release Card: Bifrost Gateway v0.1.0-test

**Release Date:** 2025-07-08  
**Release Type:** ALPHA

## 🚀 Protocol Support

### MODBUS ✅
- **Status:** stable
- **Version:** 1.1b3
- **Limitations:** No RTU over TCP support

### OPCUA 🚧
- **Status:** experimental
- **Version:** 0.1.0
- **Limitations:** Limited security features

### ETHERNETIP 🚧
- **Status:** experimental
- **Version:** 0.1.0
- **Limitations:** Basic read/write only

## 📊 Performance Metrics

- **Throughput:** 18,500 ops/sec
- **Latency (P95):** 1.2ms
- **Memory Usage:** 42MB
- **Overall Score:** 88/100

## 🧪 Testing Coverage

### Virtual Device Tests
- **Total Tests:** 25
- **Passed:** 22
- **Failed:** 3

### Go Gateway Tests  
- **Total Tests:** 45
- **Coverage:** 78%

## ✅ Quality Gates

- **Test Coverage:** ✅
- **Performance Targets:** ✅
- **Documentation Complete:** ✅
- **Approved For Release:** ✅

## 📋 Installation

```bash
# Download and install
wget https://github.com/IamMikeHelsel/bifrost/releases/download/v0.1.0-test/bifrost-gateway-linux-amd64
chmod +x bifrost-gateway-linux-amd64
./bifrost-gateway-linux-amd64
```

## 🔗 Resources

- [Documentation](https://github.com/IamMikeHelsel/bifrost/blob/main/README.md)
- [Performance Details](https://github.com/IamMikeHelsel/bifrost/blob/main/go-gateway/PERFORMANCE_OPTIMIZATIONS.md)
- [Production Deployment](https://github.com/IamMikeHelsel/bifrost/blob/main/go-gateway/docs/runbooks/production-deployment.md)
