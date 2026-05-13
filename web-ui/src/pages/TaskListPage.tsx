import React from 'react';
import { Spin } from 'antd';
import { TaskList } from '../components/TaskList';
import { useTaskStore } from '../stores/taskStore';

export const TaskListPage: React.FC = () => {
  const loading = useTaskStore(s => s.loading);

  return (
    <Spin spinning={loading}>
      <TaskList />
    </Spin>
  );
};