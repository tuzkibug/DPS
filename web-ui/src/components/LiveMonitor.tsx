import React, { useEffect, useState } from 'react';
import { Card, Button, Space, Popconfirm, message } from 'antd';
import type { TaskStats } from '../api/types';
import { useTaskStore } from '../stores/taskStore';
import { WS_URL } from '../api';

const fmtDate = (s: string | null | undefined) => {
  if (!s) return '-';
  return new Date(s).toLocaleString();
};

const fmtDuration = (ms: number) => {
  if (ms <= 0) return '0s';
  const totalSec = Math.floor(ms / 1000);
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  if (h > 0) return `${h}h ${m}m ${s}s`;
  if (m > 0) return `${m}m ${s}s`;
  return `${s}s`;
};

interface LiveMonitorProps {
  taskId: string;
  onDeleted?: () => void;
}

export const LiveMonitor: React.FC<LiveMonitorProps> = ({ taskId, onDeleted }) => {
  const [stats, setStats] = useState<TaskStats | null>(null);
  const task = useTaskStore(s => s.tasks.find(t => t.id === taskId));
  const startTask = useTaskStore(s => s.startTask);
  const stopTask = useTaskStore(s => s.stopTask);
  const deleteTask = useTaskStore(s => s.deleteTask);
  const pushQpsSample = useTaskStore(s => s.pushQpsSample);
  const setTaskStatus = useTaskStore(s => s.setTaskStatus);

  useEffect(() => {
    const ws = new WebSocket(`${WS_URL}/${taskId}`);

    ws.onopen = () => {
      setStats(null); // reset for fresh connection
    };
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'stats') {
          const st = msg.data as TaskStats;
          setStats(st);
          pushQpsSample(taskId, st.current_qps, Date.now());
          // Sync store status so buttons react to auto-stop
          const cur = useTaskStore.getState().tasks.find(t => t.id === taskId);
          if (cur && cur.status !== st.status) {
            setTaskStatus(taskId, st.status);
          }
        }
        if (msg.type === 'status_change') {
          const newStatus = msg.data.status;
          setTaskStatus(taskId, newStatus);
          if (newStatus !== 'running') {
            setStats(prev => prev ? { ...prev, current_qps: 0, status: newStatus } : null);
            pushQpsSample(taskId, 0, Date.now());
          } else {
            setStats(prev => prev ? { ...prev, status: newStatus } : null);
          }
        }
      } catch (e) {
        console.error('Failed to parse WS message', e);
      }
    };
    ws.onerror = () => {
      console.error('WebSocket connection error');
    };
    ws.onclose = () => {
      setStats(null);
    };

    return () => ws.close();
  }, [taskId]);

  const handleStart = async () => {
    try {
      await startTask(taskId);
    } catch (e) {
      message.error((e as Error).message || 'Failed to start task');
    }
  };
  const handleStop = async () => {
    try {
      await stopTask(taskId);
    } catch (e) {
      message.error((e as Error).message || 'Failed to stop task');
    }
  };
  const handleDelete = () => {
    deleteTask(taskId);
    onDeleted?.();
  };

  const isRunning = task?.status === 'running';

  return (
    <Card
      title={task?.name || taskId}
      extra={
        <Space>
          {isRunning ? (
            <Popconfirm
              title="Stop this task?"
              description="Running progress will be lost."
              onConfirm={handleStop}
              okText="Stop"
              cancelText="Cancel"
              okButtonProps={{ danger: true }}
            >
              <Button danger>Stop</Button>
            </Popconfirm>
          ) : (
            <Button type="primary" onClick={handleStart}>Start</Button>
          )}
          <Popconfirm
            title="Delete this task?"
            onConfirm={handleDelete}
            okText="Yes"
            cancelText="No"
          >
            <Button danger>Delete</Button>
          </Popconfirm>
        </Space>
      }
    >
      {stats && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 16 }}>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.current_qps.toFixed(2)}</div>
            <div>Current QPS</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.failed_count}</div>
            <div>Failed</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{fmtDuration(stats.elapsed_ms)}</div>
            <div>Current Run Time</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 14, fontWeight: 'bold' }}>{fmtDate(stats.created_at)}</div>
            <div>Created</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 14, fontWeight: 'bold' }}>{fmtDate(stats.last_run_at)}</div>
            <div>Last Run</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{fmtDuration(stats.total_run_ms)}</div>
            <div>Total Run Time</div>
          </Card>
        </div>
      )}
      {!stats && (
        <div style={{ color: '#999' }}>No statistics available</div>
      )}
    </Card>
  );
};
