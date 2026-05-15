import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Table, Tag, Space, Button, Popconfirm, message } from 'antd';
import { ReloadOutlined, PlayCircleOutlined, PauseCircleOutlined, DeleteOutlined } from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Task } from '../api/types';
import { useTaskStore } from '../stores/taskStore';
import { LiveMonitor } from './LiveMonitor';
import { QpsSparkline } from './QpsSparkline';

const statusColors: Record<string, string> = {
  pending: 'default',
  running: 'processing',
  stopped: 'warning',
  completed: 'success',
  failed: 'error',
};

export const TaskList: React.FC = () => {
  const { tasks, fetchTasks, batchStart, batchStop, batchDelete } = useTaskStore();
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const handleRefresh = useCallback(() => {
    fetchTasks();
    setSelectedRowKeys([]);
  }, [fetchTasks]);

  const selectedIds = useMemo(() => selectedRowKeys.map(String), [selectedRowKeys]);
  const selectedTasks = useMemo(
    () => tasks.filter(t => selectedIds.includes(t.id)),
    [tasks, selectedIds],
  );
  const runnableCount = useMemo(
    () => selectedTasks.filter(t => t.status !== 'running').length,
    [selectedTasks],
  );
  const runningCount = useMemo(
    () => selectedTasks.filter(t => t.status === 'running').length,
    [selectedTasks],
  );

  const handleBatchStart = async () => {
    const ids = selectedTasks.filter(t => t.status !== 'running').map(t => t.id);
    if (ids.length === 0) return;
    try {
      await batchStart(ids);
      message.success(`Started ${ids.length} task(s)`);
      setSelectedRowKeys([]);
    } catch (e) {
      message.error((e as Error).message || 'Batch start failed');
    }
  };

  const handleBatchStop = async () => {
    const ids = selectedTasks.filter(t => t.status === 'running').map(t => t.id);
    if (ids.length === 0) return;
    try {
      await batchStop(ids);
      message.success(`Stopped ${ids.length} task(s)`);
      setSelectedRowKeys([]);
    } catch (e) {
      message.error((e as Error).message || 'Batch stop failed');
    }
  };

  const handleBatchDelete = async () => {
    try {
      await batchDelete(selectedIds);
      message.success(`Deleted ${selectedIds.length} task(s)`);
      setSelectedRowKeys([]);
    } catch (e) {
      message.error((e as Error).message || 'Batch delete failed');
    }
  };

  const columns: ColumnsType<Task> = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, task: Task) => <Link to={`/tasks/${task.id}`}>{name}</Link>,
    },
    { title: 'Type', dataIndex: 'input_type', key: 'input_type' },
    { title: 'Destination', dataIndex: 'dst_ip', key: 'dst_ip' },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => <Tag color={statusColors[status]}>{status}</Tag>,
    },
    {
      title: 'QPS',
      key: 'qps',
      width: 220,
      render: (_, task) => <QpsSparkline taskId={task.id} />,
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, task) => (
        <Space>
          <LiveMonitor taskId={task.id} />
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Space>
          {selectedRowKeys.length > 0 && (
            <>
              <span style={{ marginRight: 8 }}>{selectedRowKeys.length} selected</span>
              {runnableCount > 0 && (
                <Popconfirm
                  title={`Start ${runnableCount} task(s)?`}
                  onConfirm={handleBatchStart}
                  okText="Start"
                  cancelText="Cancel"
                >
                  <Button icon={<PlayCircleOutlined />}>Start ({runnableCount})</Button>
                </Popconfirm>
              )}
              {runningCount > 0 && (
                <Popconfirm
                  title={`Stop ${runningCount} task(s)?`}
                  description="Running progress will be lost."
                  onConfirm={handleBatchStop}
                  okText="Stop"
                  cancelText="Cancel"
                  okButtonProps={{ danger: true }}
                >
                  <Button danger icon={<PauseCircleOutlined />}>Stop ({runningCount})</Button>
                </Popconfirm>
              )}
              <Popconfirm
                title={`Delete ${selectedRowKeys.length} task(s)?`}
                description="This action cannot be undone."
                onConfirm={handleBatchDelete}
                okText="Delete"
                cancelText="Cancel"
                okButtonProps={{ danger: true }}
              >
                <Button danger icon={<DeleteOutlined />}>Delete ({selectedRowKeys.length})</Button>
              </Popconfirm>
            </>
          )}
        </Space>
        <Button icon={<ReloadOutlined />} onClick={handleRefresh}>Refresh</Button>
      </div>
      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={tasks}
        rowKey="id"
        pagination={false}
        locale={{ emptyText: 'No tasks yet. Create one!' }}
      />
    </div>
  );
};
