// Ladder Logic Visualization Engine
(function() {
    'use strict';

    // Global variables
    let ladderProgram = null;
    let realTimeEnabled = false;
    let currentZoom = 1;
    let paper = null;
    let graph = null;
    const vscode = acquireVsCodeApi();

    // Initialize the ladder logic viewer
    function init() {
        console.log('Initializing Ladder Logic Viewer');
        
        // Set up JointJS paper and graph
        setupJointJS();
        
        // Set up event handlers
        setupEventHandlers();
        
        // Request initial program data
        loadSampleProgram();
    }

    function setupJointJS() {
        // Create JointJS graph and paper
        graph = new joint.dia.Graph();
        
        paper = new joint.dia.Paper({
            el: document.getElementById('ladder-diagram'),
            model: graph,
            width: '100%',
            height: '100%',
            gridSize: 10,
            drawGrid: true,
            background: {
                color: 'transparent'
            },
            defaultAnchor: { name: 'perpendicular' },
            defaultConnectionPoint: { name: 'boundary' },
            interactive: {
                vertexAdd: false,
                vertexRemove: false,
                arrowheadMove: false
            }
        });

        // Set up paper event handlers
        paper.on('cell:pointerclick', function(cellView) {
            const cell = cellView.model;
            console.log('Clicked on element:', cell.get('ladderData'));
            
            // Show element properties or toggle state for testing
            if (cell.get('ladderData')) {
                showElementInfo(cell.get('ladderData'));
            }
        });
    }

    function setupEventHandlers() {
        // Toolbar buttons
        document.getElementById('load-program')?.addEventListener('click', () => {
            vscode.postMessage({ command: 'loadProgram' });
        });

        document.getElementById('export-diagram')?.addEventListener('click', () => {
            vscode.postMessage({ command: 'exportDiagram' });
        });

        document.getElementById('toggle-realtime')?.addEventListener('click', () => {
            realTimeEnabled = !realTimeEnabled;
            document.getElementById('toggle-realtime').textContent = 
                realTimeEnabled ? 'Stop Real-time' : 'Real-time Monitor';
            vscode.postMessage({ command: 'toggleRealTime' });
        });

        // Zoom controls (will be added dynamically)
        addZoomControls();

        // Listen for messages from the extension
        window.addEventListener('message', event => {
            const message = event.data;
            switch (message.command) {
                case 'updateProgram':
                    loadProgram(message.program);
                    break;
                case 'updateRealTimeData':
                    updateRealTimeData(message.data);
                    break;
            }
        });
    }

    function addZoomControls() {
        const container = document.querySelector('.ladder-diagram-container');
        const zoomControls = document.createElement('div');
        zoomControls.className = 'zoom-controls';
        zoomControls.innerHTML = `
            <button class="zoom-button" id="zoom-in">+</button>
            <button class="zoom-button" id="zoom-out">−</button>
            <button class="zoom-button" id="zoom-fit">⌂</button>
        `;
        container.appendChild(zoomControls);

        document.getElementById('zoom-in')?.addEventListener('click', () => zoomIn());
        document.getElementById('zoom-out')?.addEventListener('click', () => zoomOut());
        document.getElementById('zoom-fit')?.addEventListener('click', () => zoomToFit());
    }

    function loadSampleProgram() {
        const sampleProgram = {
            rungs: [
                {
                    id: 'rung_001',
                    elements: [
                        { type: 'contact', id: 'input_001', address: 'I:0/0', position: { x: 100, y: 50 }, state: true, normally: 'open' },
                        { type: 'contact', id: 'input_002', address: 'I:0/1', position: { x: 200, y: 50 }, state: false, normally: 'open' },
                        { type: 'coil', id: 'output_001', address: 'O:0/0', position: { x: 350, y: 50 }, state: true }
                    ],
                    connections: [
                        { from: 'input_001', to: 'input_002' },
                        { from: 'input_002', to: 'output_001' }
                    ]
                },
                {
                    id: 'rung_002', 
                    elements: [
                        { type: 'contact', id: 'input_003', address: 'I:0/2', position: { x: 100, y: 150 }, state: false, normally: 'closed' },
                        { type: 'timer', id: 'timer_001', address: 'T4:0', position: { x: 200, y: 150 }, preset: 5000, accumulated: 1250, timing: true },
                        { type: 'coil', id: 'output_002', address: 'O:0/1', position: { x: 350, y: 150 }, state: false }
                    ],
                    connections: [
                        { from: 'input_003', to: 'timer_001' },
                        { from: 'timer_001', to: 'output_002' }
                    ]
                }
            ]
        };
        loadProgram(sampleProgram);
    }

    function loadProgram(program) {
        ladderProgram = program;
        graph.clear();
        
        console.log('Loading ladder logic program:', program);
        
        // Render each rung
        program.rungs.forEach((rung, index) => {
            renderRung(rung, index);
        });

        // Update status bar
        updateStatusBar();
        
        // Zoom to fit content
        setTimeout(() => zoomToFit(), 100);
    }

    function renderRung(rung, rungIndex) {
        const rungY = rungIndex * 120 + 60;
        const rungWidth = 500;
        
        // Create power rails (left and right vertical lines)
        const leftRail = createPowerRail(50, rungY - 30, 60);
        const rightRail = createPowerRail(rungWidth + 50, rungY - 30, 60);
        graph.addCell([leftRail, rightRail]);

        // Render elements
        const elementCells = {};
        rung.elements.forEach(element => {
            const cell = createElement(element, rungY);
            elementCells[element.id] = cell;
            graph.addCell(cell);
        });

        // Create connections between elements
        rung.connections.forEach(connection => {
            const fromElement = elementCells[connection.from];
            const toElement = elementCells[connection.to];
            
            if (fromElement && toElement) {
                const link = createConnection(fromElement, toElement, rung);
                graph.addCell(link);
            }
        });

        // Connect first element to left rail
        if (rung.elements.length > 0) {
            const firstElement = elementCells[rung.elements[0].id];
            const leftConnection = createRailConnection(leftRail, firstElement);
            graph.addCell(leftConnection);
        }

        // Connect last element to right rail
        if (rung.elements.length > 0) {
            const lastElement = elementCells[rung.elements[rung.elements.length - 1].id];
            const rightConnection = createRailConnection(lastElement, rightRail);
            graph.addCell(rightConnection);
        }

        // Add rung number label
        const rungLabel = createRungLabel(rungIndex + 1, 20, rungY);
        graph.addCell(rungLabel);
    }

    function createElement(element, rungY) {
        let cell;
        const position = { x: element.position.x, y: rungY - 10 };

        switch (element.type) {
            case 'contact':
                cell = createContact(element, position);
                break;
            case 'coil':
                cell = createCoil(element, position);
                break;
            case 'timer':
                cell = createTimer(element, position);
                break;
            default:
                console.warn('Unknown element type:', element.type);
                return null;
        }

        if (cell) {
            cell.set('ladderData', element);
        }

        return cell;
    }

    function createContact(element, position) {
        const color = element.state ? '#4CAF50' : '#757575';
        const strokeColor = element.state ? '#2E7D32' : '#424242';
        
        const contact = new joint.shapes.standard.Rectangle({
            position: position,
            size: { width: 40, height: 20 },
            attrs: {
                body: {
                    fill: color,
                    stroke: strokeColor,
                    strokeWidth: 2
                },
                label: {
                    text: element.normally === 'closed' ? '/' : '',
                    fontSize: 14,
                    fontWeight: 'bold',
                    fill: '#ffffff'
                }
            }
        });

        // Add address label
        const addressLabel = new joint.shapes.standard.TextBlock({
            position: { x: position.x - 5, y: position.y + 25 },
            size: { width: 50, height: 15 },
            attrs: {
                body: { fill: 'transparent' },
                label: {
                    text: element.address,
                    fontSize: 10,
                    fill: 'var(--vscode-descriptionForeground)'
                }
            }
        });

        graph.addCell(addressLabel);
        return contact;
    }

    function createCoil(element, position) {
        const color = element.state ? '#F44336' : '#757575';
        const strokeColor = element.state ? '#C62828' : '#424242';
        
        const coil = new joint.shapes.standard.Circle({
            position: position,
            size: { width: 40, height: 20 },
            attrs: {
                body: {
                    fill: color,
                    stroke: strokeColor,
                    strokeWidth: 2
                },
                label: {
                    text: '( )',
                    fontSize: 12,
                    fontWeight: 'bold',
                    fill: '#ffffff'
                }
            }
        });

        // Add address label
        const addressLabel = new joint.shapes.standard.TextBlock({
            position: { x: position.x - 5, y: position.y + 25 },
            size: { width: 50, height: 15 },
            attrs: {
                body: { fill: 'transparent' },
                label: {
                    text: element.address,
                    fontSize: 10,
                    fill: 'var(--vscode-descriptionForeground)'
                }
            }
        });

        graph.addCell(addressLabel);
        return coil;
    }

    function createTimer(element, position) {
        const color = element.timing ? '#FF9800' : '#757575';
        const strokeColor = element.timing ? '#F57C00' : '#424242';
        
        const timer = new joint.shapes.standard.Rectangle({
            position: position,
            size: { width: 80, height: 40 },
            attrs: {
                body: {
                    fill: color,
                    stroke: strokeColor,
                    strokeWidth: 2
                },
                label: {
                    text: `TON\n${element.accumulated}/${element.preset}`,
                    fontSize: 10,
                    fontWeight: 'bold',
                    fill: '#ffffff'
                }
            }
        });

        // Add address label
        const addressLabel = new joint.shapes.standard.TextBlock({
            position: { x: position.x + 10, y: position.y + 45 },
            size: { width: 60, height: 15 },
            attrs: {
                body: { fill: 'transparent' },
                label: {
                    text: element.address,
                    fontSize: 10,
                    fill: 'var(--vscode-descriptionForeground)'
                }
            }
        });

        graph.addCell(addressLabel);
        return timer;
    }

    function createConnection(fromElement, toElement, rung) {
        // Determine if connection is energized based on element states
        const isEnergized = shouldConnectionBeEnergized(fromElement, toElement, rung);
        
        const link = new joint.shapes.standard.Link({
            source: { id: fromElement.id },
            target: { id: toElement.id },
            attrs: {
                line: {
                    stroke: isEnergized ? '#4CAF50' : '#757575',
                    strokeWidth: isEnergized ? 3 : 2
                }
            }
        });

        return link;
    }

    function createPowerRail(x, y, height) {
        return new joint.shapes.standard.Rectangle({
            position: { x: x, y: y },
            size: { width: 4, height: height },
            attrs: {
                body: {
                    fill: '#424242',
                    stroke: 'none'
                }
            }
        });
    }

    function createRailConnection(fromElement, toElement) {
        return new joint.shapes.standard.Link({
            source: { id: fromElement.id },
            target: { id: toElement.id },
            attrs: {
                line: {
                    stroke: '#757575',
                    strokeWidth: 2
                }
            }
        });
    }

    function createRungLabel(rungNumber, x, y) {
        return new joint.shapes.standard.TextBlock({
            position: { x: x, y: y - 10 },
            size: { width: 25, height: 20 },
            attrs: {
                body: {
                    fill: '#2196F3',
                    stroke: '#1976D2',
                    strokeWidth: 1,
                    rx: 10,
                    ry: 10
                },
                label: {
                    text: rungNumber.toString(),
                    fontSize: 12,
                    fontWeight: 'bold',
                    fill: '#ffffff'
                }
            }
        });
    }

    function shouldConnectionBeEnergized(fromElement, toElement, rung) {
        // Simple logic for demonstration - in real implementation this would
        // be based on actual PLC logic evaluation
        const fromData = fromElement.get('ladderData');
        const toData = toElement.get('ladderData');
        
        if (fromData && fromData.state) {
            return true;
        }
        
        return false;
    }

    function updateRealTimeData(data) {
        if (!realTimeEnabled || !ladderProgram) return;
        
        // Update element states based on real-time data
        // This would integrate with the actual PLC data
        console.log('Updating real-time data:', data);
        
        // Refresh the display
        loadProgram(ladderProgram);
    }

    function showElementInfo(elementData) {
        vscode.postMessage({
            command: 'alert',
            text: `Element: ${elementData.address}\nType: ${elementData.type}\nState: ${elementData.state ? 'ON' : 'OFF'}`
        });
    }

    function updateStatusBar() {
        if (!ladderProgram) return;
        
        const rungCount = ladderProgram.rungs.length;
        const elementCount = ladderProgram.rungs.reduce((total, rung) => total + rung.elements.length, 0);
        
        document.getElementById('rung-count').textContent = rungCount;
        document.getElementById('element-count').textContent = elementCount;
        document.getElementById('connection-status').textContent = realTimeEnabled ? 'Live' : 'Static';
    }

    function zoomIn() {
        currentZoom *= 1.2;
        paper.scale(currentZoom);
    }

    function zoomOut() {
        currentZoom /= 1.2;
        paper.scale(currentZoom);
    }

    function zoomToFit() {
        if (graph.getCells().length === 0) return;
        
        paper.scaleContentToFit({
            padding: 50,
            maxScale: 2,
            minScale: 0.2
        });
        
        currentZoom = paper.scale().sx;
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();