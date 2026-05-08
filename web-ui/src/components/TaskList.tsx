import React, { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Table, Tag, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { Task } from '../api/types';
import { useTaskStore } from '../stores/taskStore';
import { LiveMonitor } from './LiveMonitor';

const statusColors: Record<string, string> = {
  pending: 'default',
  running: 'processing',
  stopped: 'warning',
  completed: 'success',
  failed: 'error',
};

export const TaskList: React.FC = () => {
  const { tasks, fetchTasks } = useTaskStore();

  useEffect(() => {
    fetchTasks();
  }, []);

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
      title: 'Actions',
      key: 'actions',
      render: (_, task) => (
        <Space>
          <LiveMonitor taskId={task.id} />
        </Space>
      ),
    },
  ];

  return <Table columns={columns} dataSource={tasks} rowKey="id" pagination={false} />;
};