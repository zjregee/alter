let usageChart = null;

export function setupUsageHandlers() {
    document.body.addEventListener('click', (e) => {
        const rangeBtn = e.target.closest('.usage-range-btn');
        if (rangeBtn) {
            handleRangeChange(rangeBtn);
        }
    });
}

export function loadUsageData() {
    const container = document.querySelector('.usage-container');
    if (!container) return;

    container.innerHTML = `
        <div class="usage-chart-container">
             <div class="usage-chart-header">
                <div class="usage-controls">
                    <button class="usage-range-btn active" data-range="7d">最近一周</button>
                    <button class="usage-range-btn" data-range="30d">最近一个月</button>
                    <button class="usage-range-btn" data-range="this_week">本周</button>
                    <button class="usage-range-btn" data-range="this_month">本月</button>
                </div>
            </div>
            <div class="canvas-wrapper">
                <canvas id="usage-line-chart"></canvas>
            </div>
        </div>
        <div class="usage-heatmap-container">
            Activity heat map placeholder
        </div>
    `;

    renderLineChart('7d');
}

function handleRangeChange(button) {
    if (button.classList.contains('active')) return;

    document.querySelectorAll('.usage-range-btn').forEach((btn) => btn.classList.remove('active'));
    button.classList.add('active');

    const range = button.dataset.range;
    updateChart(range);
}

function updateChart(range) {
    if (!usageChart) {
        renderLineChart(range);
        return;
    }
    const chartData = getChartDataForRange(range);
    usageChart.data = chartData;
    usageChart.update();
}

function getChartDataForRange(range) {
    const endDate = new Date();
    let startDate = new Date();
    const labels = [];

    switch (range) {
        case '30d':
            startDate.setDate(endDate.getDate() - 29);
            break;
        case 'this_week': {
            const day = endDate.getDay();
            const diff = endDate.getDate() - day + (day === 0 ? -6 : 1); // Adjust for Sunday
            startDate = new Date(new Date().setDate(diff));
            break;
        }
        case 'this_month':
            startDate = new Date(endDate.getFullYear(), endDate.getMonth(), 1);
            break;
        case '7d':
        default:
            startDate.setDate(endDate.getDate() - 6);
            break;
    }

    const date = new Date(startDate);
    while (date <= endDate) {
        labels.push(date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }));
        date.setDate(date.getDate() + 1);
    }

    const generateData = (count) =>
        Array.from({ length: count }, () => Math.random() * (Math.random() > 0.8 ? 15 : 5)).map(
            (v) => Math.round(v * 100) / 100
        );

    return {
        labels,
        datasets: [
            {
                label: 'gemini-1.5-pro',
                data: generateData(labels.length),
                borderColor: 'rgba(79, 130, 217, 0.8)',
                backgroundColor: 'rgba(79, 130, 217, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 2,
                pointHoverRadius: 6
            },
            {
                label: 'claude-3-opus',
                data: generateData(labels.length),
                borderColor: 'rgba(91, 214, 142, 0.8)',
                backgroundColor: 'rgba(91, 214, 142, 0.1)',
                fill: true,
                tension: 0.4,
                pointRadius: 2,
                pointHoverRadius: 6
            }
        ]
    };
}

function renderLineChart(range) {
    const ctx = document.getElementById('usage-line-chart')?.getContext('2d');
    if (!ctx) return;

    if (usageChart) {
        usageChart.destroy();
    }

    const chartData = getChartDataForRange(range);

    const options = {
        responsive: true,
        maintainAspectRatio: false,
        layout: {
            padding: {
                right: 40
            }
        },
        scales: {
            x: {
                grid: {
                    color: 'rgba(255, 255, 255, 0.05)'
                },
                ticks: {
                    color: '#9aa3b2',
                    font: {
                        size: 10
                    }
                }
            },
            y: {
                beginAtZero: true,
                grid: {
                    color: 'rgba(255, 255, 255, 0.05)'
                },
                ticks: {
                    color: '#9aa3b2',
                    callback: (value) => `$${value.toFixed(2)}`
                }
            }
        },
        plugins: {
            legend: {
                display: true,
                position: 'right',
                align: 'end',
                labels: {
                    color: '#d6dbe4',
                    padding: 20,
                    boxWidth: 10,
                    boxHeight: 10,
                    font: {
                        size: 11
                    },
                    generateLabels: (chart) => {
                        const labels = window.Chart.defaults.plugins.legend.labels.generateLabels(chart);
                        labels.forEach((label) => {
                            const dataset = chart.data.datasets[label.datasetIndex];
                            if (dataset?.borderColor) {
                                label.fillStyle = dataset.borderColor;
                                label.strokeStyle = dataset.borderColor;
                                label.lineWidth = 0;
                            }
                        });
                        return labels;
                    }
                }
            },
            tooltip: {
                backgroundColor: '#1e2229',
                titleColor: '#d6dbe4',
                bodyColor: '#9aa3b2',
                padding: 10,
                cornerRadius: 8,
                boxPadding: 4,
                boxWidth: 10,
                boxHeight: 10,
                borderColor: 'transparent',
                borderWidth: 0,
                usePointStyle: false,
                callbacks: {
                    labelColor: (context) => {
                        const color = context.dataset.borderColor || context.dataset.backgroundColor;
                        return {
                            borderColor: color,
                            backgroundColor: color,
                            borderWidth: 0
                        };
                    }
                }
            }
        },
        interaction: {
            intersect: false,
            mode: 'index'
        }
    };

    usageChart = new window.Chart(ctx, {
        type: 'line',
        data: chartData,
        options: options
    });
}
