import React from 'react';
import { TaskForm } from '../components/TaskForm';
import { TaskList } from '../components/TaskList';
import { Card } from 'antd';

export const TaskCreatePage: React.FC = () => {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 24 }}>
      <TaskForm />
      <Card title="Recent Tasks">
        <TaskList />
      </Card>
    </div>
  );
};