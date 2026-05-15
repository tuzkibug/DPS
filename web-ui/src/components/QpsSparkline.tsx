import React from 'react';
import { useTaskStore } from '../stores/taskStore';

interface QpsSparklineProps {
  taskId: string;
  width?: number;
  height?: number;
}

export const QpsSparkline: React.FC<QpsSparklineProps> = ({ taskId, width = 200, height = 80 }) => {
  const samples = useTaskStore(s => s.qpsSamples[taskId] ?? []);

  if (samples.length < 2) {
    return (
      <div style={{ width, height, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ccc', fontSize: 12 }}>
        -
      </div>
    );
  }

  const qpsValues = samples.map(s => s.qps);
  const maxQps = Math.max(...qpsValues, 0.001);
  const minQps = Math.min(...qpsValues);
  const range = maxQps - minQps || 1;
  const timeMin = samples[0].time;
  const timeMax = samples[samples.length - 1].time;
  const timeRange = timeMax - timeMin || 1;

  const margin = { top: 6, right: 6, bottom: 18, left: 32 };
  const plotW = width - margin.left - margin.right;
  const plotH = height - margin.top - margin.bottom;

  const toX = (t: number) => margin.left + ((t - timeMin) / timeRange) * plotW;
  const toY = (q: number) => margin.top + plotH - ((q - minQps) / range) * plotH;

  const linePath = samples
    .map((s, i) => `${i === 0 ? 'M' : 'L'}${toX(s.time).toFixed(1)},${toY(s.qps).toFixed(1)}`)
    .join(' ');

  // Y axis ticks (3 levels)
  const yTicks = [minQps, minQps + range / 2, maxQps].map(v => ({
    value: v,
    label: v >= 1000 ? `${(v / 1000).toFixed(1)}k` : v.toFixed(1),
    y: toY(v),
  }));

  // X axis ticks (3 time labels)
  const xTicks = [timeMin, timeMin + timeRange / 2, timeMax].map(t => ({
    label: new Date(t).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }),
    x: toX(t),
  }));

  return (
    <svg width={width} height={height} role="img" aria-label="QPS sparkline" style={{ display: 'block', fontFamily: 'monospace' }}>
      {/* Grid lines */}
      {yTicks.map((t, i) => (
        <line key={`gy-${i}`} x1={margin.left} y1={t.y} x2={width - margin.right} y2={t.y} stroke="#f0f0f0" strokeWidth={0.5} />
      ))}
      {xTicks.map((t, i) => (
        <line key={`gx-${i}`} x1={t.x} y1={margin.top} x2={t.x} y2={height - margin.bottom} stroke="#f0f0f0" strokeWidth={0.5} />
      ))}

      {/* Axes */}
      <line x1={margin.left} y1={margin.top} x2={margin.left} y2={height - margin.bottom} stroke="#d9d9d9" strokeWidth={1} />
      <line x1={margin.left} y1={height - margin.bottom} x2={width - margin.right} y2={height - margin.bottom} stroke="#d9d9d9" strokeWidth={1} />

      {/* Data line */}
      <path d={linePath} fill="none" stroke="#1677ff" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />

      {/* Y axis labels */}
      {yTicks.map((t, i) => (
        <text key={`yl-${i}`} x={margin.left - 4} y={t.y + 4} textAnchor="end" fontSize={9} fill="#999">
          {t.label}
        </text>
      ))}

      {/* X axis labels */}
      {xTicks.map((t, i) => (
        <text key={`xl-${i}`} x={t.x} y={height - 2} textAnchor={i === 0 ? 'start' : i === 2 ? 'end' : 'middle'} fontSize={9} fill="#999">
          {t.label}
        </text>
      ))}
    </svg>
  );
};
