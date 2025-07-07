#!/usr/bin/env node

/**
 * Check if TypeScript-Go is available and fallback gracefully
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

function checkTypescriptGo() {
    console.log('🔍 Checking for TypeScript-Go compiler...');
    
    try {
        // Check if tsc-go is available
        execSync('tsc-go --version', { stdio: 'pipe' });
        console.log('✅ TypeScript-Go found! Using 10x faster compilation.');
        
        // Update npm scripts to use Go compiler
        updatePackageScripts(true);
        return true;
    } catch (error) {
        console.log('⚠️  TypeScript-Go not found. Using standard TypeScript compiler.');
        console.log('💡 To enable 10x faster builds, install: npm install @typescript/go@preview');
        
        // Ensure fallback to standard TypeScript
        updatePackageScripts(false);
        return false;
    }
}

function updatePackageScripts(useGo) {
    const packagePath = path.join(__dirname, '..', 'package.json');
    const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
    
    if (useGo) {
        // Use TypeScript-Go
        pkg.scripts.compile = 'tsc-go -p ./';
        pkg.scripts.watch = 'tsc-go -watch -p ./';
        console.log('📝 Updated package.json to use TypeScript-Go');
    } else {
        // Fallback to standard TypeScript
        pkg.scripts.compile = 'tsc -p ./';
        pkg.scripts.watch = 'tsc -watch -p ./';
        console.log('📝 Updated package.json to use standard TypeScript');
    }
    
    fs.writeFileSync(packagePath, JSON.stringify(pkg, null, 2));
}

function checkPerformanceSettings() {
    const config = require('vscode').workspace.getConfiguration('bifrost');
    const useTypescriptGo = config.get('experimental.useTypescriptGo', false);
    
    if (useTypescriptGo && !checkTypescriptGo()) {
        console.log('⚠️  TypeScript-Go enabled in settings but not available');
        console.log('   Please install: npm install @typescript/go@preview');
        return false;
    }
    
    return true;
}

// Run check
const isAvailable = checkTypescriptGo();

if (isAvailable) {
    console.log('🚀 Ready for blazing fast TypeScript compilation!');
} else {
    console.log('🔧 Using standard TypeScript compilation.');
}

process.exit(0);