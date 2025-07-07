#!/usr/bin/env node

/**
 * Benchmark TypeScript vs TypeScript-Go compilation performance
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

class CompilationBenchmark {
    constructor() {
        this.results = {
            standard: { times: [], average: 0, errors: 0 },
            go: { times: [], average: 0, errors: 0 }
        };
    }
    
    async runBenchmark() {
        console.log('🏁 Starting TypeScript compilation benchmark...\n');
        
        // Clean build directory
        this.cleanBuild();
        
        // Benchmark standard TypeScript
        console.log('📊 Benchmarking standard TypeScript compiler...');
        await this.benchmarkCompiler('tsc', 'standard', 5);
        
        // Check if TypeScript-Go is available
        if (this.isTypescriptGoAvailable()) {
            console.log('\n📊 Benchmarking TypeScript-Go native compiler...');
            await this.benchmarkCompiler('npx tsgo', 'go', 5);
        } else {
            console.log('\n⚠️  TypeScript-Go not available. Skipping Go benchmark.');
            console.log('   Install with: npm install @typescript/native-preview');
        }
        
        this.displayResults();
    }
    
    cleanBuild() {
        const outDir = path.join(__dirname, '..', 'out');
        if (fs.existsSync(outDir)) {
            fs.rmSync(outDir, { recursive: true, force: true });
        }
    }
    
    isTypescriptGoAvailable() {
        try {
            execSync('npx tsgo --version', { stdio: 'pipe' });
            return true;
        } catch {
            return false;
        }
    }
    
    async benchmarkCompiler(compiler, type, iterations) {
        for (let i = 1; i <= iterations; i++) {
            console.log(`  Run ${i}/${iterations}...`);
            
            try {
                this.cleanBuild(); // Fresh build each time
                
                const start = performance.now();
                execSync(`${compiler} -p ./`, { 
                    cwd: path.join(__dirname, '..'),
                    stdio: 'pipe' 
                });
                const end = performance.now();
                
                const duration = end - start;
                this.results[type].times.push(duration);
                
                console.log(`    ✅ ${duration.toFixed(2)}ms`);
                
            } catch (error) {
                console.log(`    ❌ Error: ${error.message}`);
                this.results[type].errors++;
            }
            
            // Small delay between runs
            await new Promise(resolve => setTimeout(resolve, 100));
        }
        
        if (this.results[type].times.length > 0) {
            this.results[type].average = 
                this.results[type].times.reduce((a, b) => a + b, 0) / 
                this.results[type].times.length;
        }
    }
    
    displayResults() {
        console.log('\n📊 BENCHMARK RESULTS');
        console.log('═'.repeat(50));
        
        const standard = this.results.standard;
        const go = this.results.go;
        
        console.log('\n🔸 Standard TypeScript Compiler:');
        if (standard.times.length > 0) {
            console.log(`   Average: ${standard.average.toFixed(2)}ms`);
            console.log(`   Min:     ${Math.min(...standard.times).toFixed(2)}ms`);
            console.log(`   Max:     ${Math.max(...standard.times).toFixed(2)}ms`);
            console.log(`   Runs:    ${standard.times.length}/${standard.times.length + standard.errors}`);
        } else {
            console.log('   ❌ No successful runs');
        }
        
        console.log('\n🔸 TypeScript-Go Compiler:');
        if (go.times.length > 0) {
            console.log(`   Average: ${go.average.toFixed(2)}ms`);
            console.log(`   Min:     ${Math.min(...go.times).toFixed(2)}ms`);
            console.log(`   Max:     ${Math.max(...go.times).toFixed(2)}ms`);
            console.log(`   Runs:    ${go.times.length}/${go.times.length + go.errors}`);
            
            // Calculate speedup
            if (standard.average > 0) {
                const speedup = standard.average / go.average;
                console.log(`\n🚀 SPEEDUP: ${speedup.toFixed(1)}x faster!`);
                
                if (speedup >= 10) {
                    console.log('   🎉 Achieved 10x+ performance improvement!');
                } else if (speedup >= 5) {
                    console.log('   ⚡ Significant performance improvement!');
                } else if (speedup >= 2) {
                    console.log('   ✨ Notable performance improvement!');
                }
            }
        } else {
            console.log('   ⚠️  Not available or failed to run');
        }
        
        console.log('\n💾 Detailed results:');
        console.log('Standard times:', standard.times.map(t => t.toFixed(2)).join(', '));
        if (go.times.length > 0) {
            console.log('Go times:      ', go.times.map(t => t.toFixed(2)).join(', '));
        }
        
        // Save results to file
        this.saveResults();
    }
    
    saveResults() {
        const resultsFile = path.join(__dirname, '..', 'benchmark-results.json');
        const results = {
            timestamp: new Date().toISOString(),
            nodeVersion: process.version,
            platform: process.platform,
            arch: process.arch,
            results: this.results
        };
        
        fs.writeFileSync(resultsFile, JSON.stringify(results, null, 2));
        console.log(`\n💾 Results saved to: ${resultsFile}`);
    }
}

// Run benchmark if called directly
if (require.main === module) {
    const benchmark = new CompilationBenchmark();
    benchmark.runBenchmark().catch(console.error);
}

module.exports = CompilationBenchmark;