import React, { useState } from 'react';
import { Form, Input, InputNumber, Select, Button, Card, Upload, message } from 'antd';
import type { CreateTaskRequest, QoSConfig } from '../api/types';
import { useTaskStore } from '../stores/taskStore';

interface TaskFormProps {
  onSuccess?: () => void;
}

export const TaskForm: React.FC<TaskFormProps> = ({ onSuccess }) => {
  const [form] = Form.useForm();
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const createTask = useTaskStore(s => s.createTask);

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      let fileContent = '';
      if (file) {
        const buffer = await file.arrayBuffer();
        const bytes = new Uint8Array(buffer);
        let binary = '';
        for (let i = 0; i < bytes.length; i++) {
          binary += String.fromCharCode(bytes[i]);
        }
        fileContent = btoa(binary);
      }

      const data: CreateTaskRequest = {
        name: values.name,
        input_type: values.input_type,
        file_content: fileContent,
        src_ip: values.src_ip,
        dst_ip: values.dst_ip,
        src_mac: values.src_mac,
        dst_mac: values.dst_mac,
        start_time: values.start_time,
        duration_ms: values.duration_ms,
        qos: {
          target_qps: values.target_qps,
          jitter: values.jitter,
          delay_min_ms: values.delay_min_ms,
          delay_max_ms: values.delay_max_ms,
        } as QoSConfig,
      };

      await createTask(data);
      message.success('Task created successfully');
      form.resetFields();
      setFile(null);
      onSuccess?.();
    } catch (e) {
      message.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card title="Create DNS Send Task">
      <Form form={form} layout="vertical" onFinish={handleSubmit}>
        <Form.Item name="name" label="Task Name" rules={[{ required: true }]}>
          <Input placeholder="My DNS Task" />
        </Form.Item>

        <Form.Item name="input_type" label="Input Type" rules={[{ required: true }]}>
          <Select>
            <Select.Option value="csv">CSV (Domain List)</Select.Option>
            <Select.Option value="pcap">PCAP (Packet File)</Select.Option>
          </Select>
        </Form.Item>

        <Form.Item label="Upload File">
          <Upload beforeUpload={f => { setFile(f); return false; }} accept=".csv,.pcap">
            <Button>{file ? file.name : 'Click to Upload'}</Button>
          </Upload>
        </Form.Item>

        <Form.Item name="src_ip" label="Source IP" rules={[{ required: true }]}>
          <Input placeholder="192.168.1.100" />
        </Form.Item>

        <Form.Item name="dst_ip" label="Destination IP" rules={[{ required: true }]}>
          <Input placeholder="8.8.8.8" />
        </Form.Item>

        <Form.Item name="src_mac" label="Source MAC" rules={[{ required: true }]}>
          <Input placeholder="aa:bb:cc:dd:ee:ff" />
        </Form.Item>

        <Form.Item name="dst_mac" label="Destination MAC" rules={[{ required: true }]}>
          <Input placeholder="11:22:33:44:55:66" />
        </Form.Item>

        <Form.Item name="target_qps" label="Target QPS" initialValue={100}>
          <InputNumber min={1} max={100000} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="jitter" label="Jitter (0-1)" initialValue={0}>
          <InputNumber min={0} max={1} step={0.1} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="delay_min_ms" label="Min Delay (ms)" initialValue={0}>
          <InputNumber min={0} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item name="delay_max_ms" label="Max Delay (ms)" initialValue={0}>
          <InputNumber min={0} style={{ width: '100%' }} />
        </Form.Item>

        <Form.Item>
          <Button type="primary" htmlType="submit" loading={loading}>
            Create Task
          </Button>
        </Form.Item>
      </Form>
    </Card>
  );
};