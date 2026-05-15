import React, { useEffect, useState } from 'react';
import { LiveMonitor } from '../components/LiveMonitor';
import { EditTaskModal } from '../components/EditTaskModal';
import { useParams } from 'react-router-dom';
import { useTaskStore } from '../stores/taskStore';
import { Spin, Descriptions, Card, Typography, Button } from 'antd';

const { Link: AntLink } = Typography;

const API_BASE = '/api/v1';

export const TaskDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const { currentTask, fetchTask, loading } = useTaskStore();
  const [editOpen, setEditOpen] = useState(false);

  useEffect(() => {
    if (id) fetchTask(id);
  }, [id]);

  if (loading) {
    return <Spin tip="Loading..." />;
  }
  if (!currentTask) {
    return <div style={{ color: '#999', textAlign: 'center', padding: 48 }}>Task not found or has been deleted.</div>;
  }

  const task = currentTask;
  const fileName = task.file_path ? task.file_path.split('/').pop() : '-';
  const canEdit = task.status !== 'running';

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      <LiveMonitor taskId={id!} />

      <Card
        title="Task Configuration"
        extra={canEdit ? <Button onClick={() => setEditOpen(true)}>Edit</Button> : null}
      >
        <Descriptions bordered column={2} size="small">
          <Descriptions.Item label="Task ID">{task.id}</Descriptions.Item>
          <Descriptions.Item label="Name">{task.name}</Descriptions.Item>
          <Descriptions.Item label="Input Type">{task.input_type}</Descriptions.Item>
          <Descriptions.Item label="Status">{task.status}</Descriptions.Item>
          <Descriptions.Item label="Source IP">{task.src_ip}</Descriptions.Item>
          <Descriptions.Item label="Destination IP">{task.dst_ip}</Descriptions.Item>
          <Descriptions.Item label="Source MAC">{task.src_mac}</Descriptions.Item>
          <Descriptions.Item label="Destination MAC">{task.dst_mac}</Descriptions.Item>
          <Descriptions.Item label="Random Source IP">{task.random_src_ip ? 'Yes' : 'No'}</Descriptions.Item>
          <Descriptions.Item label="Random Source MAC">{task.random_src_mac ? 'Yes' : 'No'}</Descriptions.Item>
          {task.interface && (
            <Descriptions.Item label="Interface">{task.interface}</Descriptions.Item>
          )}
          <Descriptions.Item label="Target QPS">{task.qos.target_qps}</Descriptions.Item>
          <Descriptions.Item label="Jitter">{task.qos.jitter}</Descriptions.Item>
          <Descriptions.Item label="Min Delay (ms)">{task.qos.delay_min_ms}</Descriptions.Item>
          <Descriptions.Item label="Max Delay (ms)">{task.qos.delay_max_ms}</Descriptions.Item>
          <Descriptions.Item label="Duration (ms)">{task.duration_ms || '∞'}</Descriptions.Item>
          <Descriptions.Item label="Created At">{new Date(task.created_at).toLocaleString()}</Descriptions.Item>
          <Descriptions.Item label="File" span={2}>
            {task.file_path ? (
              task.input_type === 'pcap' ? (
                <span>{task.file_path}</span>
              ) : (
                <AntLink href={`${API_BASE}/tasks/${task.id}/file`} target="_blank">
                  {fileName}
                </AntLink>
              )
            ) : (
              '-'
            )}
          </Descriptions.Item>
        </Descriptions>
      </Card>

      <EditTaskModal
        task={task}
        open={editOpen}
        onClose={() => setEditOpen(false)}
      />
    </div>
  );
};
