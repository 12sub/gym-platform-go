(function () {
  const palette = ['#4f46e5', '#0ea5e9', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#64748b'];

  const naira = (v) =>
    '₦' + Number(v).toLocaleString('en-NG', { minimumFractionDigits: 2, maximumFractionDigits: 2 });

  function draw(id, config) {
    const canvas = document.getElementById(id);
    if (!canvas) return;
    const existing = Chart.getChart(canvas);
    if (existing) existing.destroy();
    new Chart(canvas, config);
  }

  function doughnut(labels, values) {
    return {
      type: 'doughnut',
      data: {
        labels: labels,
        datasets: [{ data: values, backgroundColor: palette, borderWidth: 0 }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { position: 'bottom' },
          tooltip: { callbacks: { label: (c) => c.label + ': ' + naira(c.parsed) } },
        },
      },
    };
  }

  function init() {
    const el = document.getElementById('revenue-data');
    if (!el || typeof Chart === 'undefined') return;
    const d = JSON.parse(el.textContent);

    draw('chart-time', {
      type: 'bar',
      data: {
        labels: d.timeLabels,
        datasets: [
          { label: d.timeTitle, data: d.timeValues, backgroundColor: palette[0], borderRadius: 4 },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { display: false },
          tooltip: { callbacks: { label: (c) => naira(c.parsed.y) } },
        },
        scales: {
          y: {
            beginAtZero: true,
            ticks: { callback: (v) => '₦' + Number(v).toLocaleString('en-NG') },
          },
        },
      },
    });

    draw('chart-plan', doughnut(d.planLabels, d.planValues));
    draw('chart-method', doughnut(d.methodLabels, d.methodValues));
  }

  document.addEventListener('DOMContentLoaded', init);
  document.addEventListener('htmx:afterSwap', init);
  init();
})();