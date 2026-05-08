import React, { useEffect, useState } from 'react';
import { Card, Button, Space, Popconfirm } from 'antd';
import type { TaskStats } from '../api/types';
import { useTaskStore } from '../stores/taskStore';
import { WS_URL } from '../api';

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

  useEffect(() => {
    const ws = new WebSocket(`${WS_URL}/${taskId}`);

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'stats') {
          setStats(msg.data);
        }
        if (msg.type === 'status_change') {
          setStats(prev => prev ? { ...prev, status: msg.data.status } : null);
        }
      } catch (e) {
        console.error('Failed to parse WS message', e);
      }
    };

    return () => ws.close();
  }, [taskId]);

  const handleStart = () => startTask(taskId);
  const handleStop = () => stopTask(taskId);
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
            <Button danger onClick={handleStop}>Stop</Button>
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
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16 }}>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.sent_count}</div>
            <div>Sent Packets</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.current_qps.toFixed(2)}</div>
            <div>Current QPS</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{stats.failed_count}</div>
            <div>Failed</div>
          </Card>
          <Card size="small">
            <div style={{ fontSize: 24, fontWeight: 'bold' }}>{Math.floor(stats.elapsed_ms / 1000)}s</div>
            <div>Elapsed</div>
          </Card>
        </div>
      )}
      {!stats && (
        <div style={{ color: '#999' }}>No statistics available</div>
      )}
    </Card>
  );
};