// Chart instances
const charts = new Map();

// Initialize charts for tags
function initializeCharts() {
    if (!device.tags) return;
    
    device.tags.forEach(tag => {
        const canvas = document.getElementById(`chart-${tag.id}`);
        if (canvas) {
            const ctx = canvas.getContext('2d');
            const chart = new Chart(ctx, {
                type: 'line',
                data: {
                    labels: [],
                    datasets: [{
                        label: tag.name,
                        data: [],
                        borderColor: getComputedStyle(document.body).getPropertyValue('--industrial-primary') || '#10b981',
                        backgroundColor: 'transparent',
                        borderWidth: 2,
                        pointRadius: 0,
                        tension: 0.1
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            display: false
                        },
                        tooltip: {
                            mode: 'index',
                            intersect: false,
                        }
                    },
                    scales: {
                        x: {
                            type: 'time',
                            time: {
                                parser: 'HH:mm:ss',
                                displayFormats: {
                                    second: 'HH:mm:ss'
                                }
                            },
                            grid: {
                                display: false
                            }
                        },
                        y: {
                            grid: {
                                color: 'rgba(255, 255, 255, 0.1)'
                            }
                        }
                    }
                }
            });
            
            charts.set(tag.id, chart);
        }
    });
}

// Update tag data and chart
function updateTagData(tag, data) {
    // Update displayed value
    const valueElement = document.querySelector(`[data-tag-id="${tag.id}"] .value`);
    if (valueElement) {
        let displayValue = tag.value;
        if (typeof tag.value === 'number') {
            displayValue = tag.value.toFixed(2);
        }
        valueElement.textContent = displayValue;
    }
    
    // Update chart
    const chart = charts.get(tag.id);
    if (chart && data) {
        const labels = data.map(d => new Date(d.time).toLocaleTimeString());
        const values = data.map(d => d.value);
        
        chart.data.labels = labels;
        chart.data.datasets[0].data = values;
        chart.update('none'); // No animation for performance
    }
}

// Initialize on load
document.addEventListener('DOMContentLoaded', () => {
    initializeCharts();
});