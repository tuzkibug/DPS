import React, { useEffect, useState } from 'react';
import { Modal, Form, Input, InputNumber, message } from 'antd';
import type { Task, UpdateTaskRequest } from '../api/types';
import { useTaskStore } from '../stores/taskStore';

interface Props {
  task: Task;
  open: boolean;
  onClose: () => void;
}

export const EditTaskModal: React.FC<Props> = ({ task, open, onClose }) => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const updateTask = useTaskStore(s => s.updateTask);

  useEffect(() => {
    if (open) {
      form.setFieldsValue({
        name: task.name,
        file_path: task.file_path,
        src_ip: task.src_ip,
        dst_ip: task.dst_ip,
        src_mac: task.src_mac,
        dst_mac: task.dst_mac,
        target_qps: task.qos.target_qps,
        jitter: task.qos.jitter,
        delay_min_ms: task.qos.delay_min_ms,
        delay_max_ms: task.qos.delay_max_ms,
        duration_sec: Math.round(task.duration_ms / 1000),
      });
    }
  }, [open, task, form]);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      setLoading(true);
      const data: UpdateTaskRequest = {
        name: values.name,
        file_path: values.file_path,
        src_ip: values.src_ip,
        dst_ip: values.dst_ip,
        src_mac: values.src_mac,
        dst_mac: values.dst_mac,
        duration_ms: values.duration_sec ? values.duration_sec * 1000 : 0,
        qos: {
          target_qps: values.target_qps,
          jitter: values.jitter,
          delay_min_ms: values.delay_min_ms,
          delay_max_ms: values.delay_max_ms,
        },
      };
      await updateTask(task.id, data);
      message.success('Task updated');
      onClose();
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return;
      message.error((e as Error).message || 'Failed to update');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal
      title="Edit Task"
      open={open}
      onOk={handleOk}
      onCancel={onClose}
      confirmLoading={loading}
      width={600}
    >
      <Form form={form} layout="vertical">
        <Form.Item name="name" label="Task Name" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="file_path" label="File Path">
          <Input placeholder="Server path for PCAP, or uploaded file path" />
        </Form.Item>
        <Form.Item name="src_ip" label="Source IP" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="dst_ip" label="Destination IP" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="src_mac" label="Source MAC" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="dst_mac" label="Destination MAC" rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        <Form.Item name="target_qps" label="Target QPS">
          <InputNumber min={1} max={100000} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="jitter" label="Jitter (0-1)">
          <InputNumber min={0} max={1} step={0.1} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="delay_min_ms" label="Min Delay (ms)">
          <InputNumber min={0} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="delay_max_ms" label="Max Delay (ms)">
          <InputNumber min={0} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item name="duration_sec" label="Duration (seconds, 0 = unlimited)">
          <InputNumber min={0} max={86400} style={{ width: '100%' }} />
        </Form.Item>
      </Form>
    </Modal>
  );
};
