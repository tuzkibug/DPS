import React, { useCallback, useEffect, useState } from 'react';
import { Form, Input, InputNumber, Select, Button, Card, Upload, message, Breadcrumb, List, Typography, Spin, Space } from 'antd';
import { FolderOutlined, FileOutlined, HomeOutlined } from '@ant-design/icons';
import type { CreateTaskRequest, QoSConfig, PcapDirList } from '../api/types';
import { useTaskStore } from '../stores/taskStore';
import { taskAPI } from '../api';

const { Text } = Typography;

interface TaskFormProps {
  onSuccess?: () => void;
}

export const TaskForm: React.FC<TaskFormProps> = ({ onSuccess }) => {
  const [form] = Form.useForm();
  const [file, setFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [inputType, setInputType] = useState<string>('csv');
  const createTask = useTaskStore(s => s.createTask);

  // PCAP directory browser state
  const [pcapPath, setPcapPath] = useState('');
  const [pcapData, setPcapData] = useState<PcapDirList | null>(null);
  const [pcapLoading, setPcapLoading] = useState(false);

  const fetchPcapDirs = useCallback(async (path?: string) => {
    setPcapLoading(true);
    try {
      const data = await taskAPI.listPcapDirs(path || undefined);
      setPcapData(data);
      setPcapPath(data.current_path);
    } catch {
      message.error('Failed to load directory');
    } finally {
      setPcapLoading(false);
    }
  }, []);

  useEffect(() => {
    if (inputType === 'pcap') {
      fetchPcapDirs('');
    }
  }, [inputType, fetchPcapDirs]);

  const navigateTo = (dir: string) => {
    const newPath = pcapPath ? `${pcapPath}/${dir}` : dir;
    fetchPcapDirs(newPath);
  };

  const navigateBreadcrumb = (index: number) => {
    const parts = pcapPath.split('/');
    const newPath = parts.slice(0, index + 1).join('/');
    fetchPcapDirs(newPath);
  };

  const handleSubmit = async (values: any) => {
    setLoading(true);
    try {
      let fileContent = '';
      let filePath = '';

      if (inputType === 'csv' || !values.input_type || values.input_type === 'csv') {
        if (file) {
          const buffer = await file.arrayBuffer();
          const bytes = new Uint8Array(buffer);
          let binary = '';
          for (let i = 0; i < bytes.length; i++) {
            binary += String.fromCharCode(bytes[i]);
          }
          fileContent = btoa(binary);
        }
      } else if (inputType === 'pcap') {
        filePath = pcapPath;
      }

      const data: CreateTaskRequest = {
        name: values.name,
        input_type: values.input_type,
        file_content: fileContent,
        file_path: filePath,
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
      setPcapData(null);
      setPcapPath('');
      onSuccess?.();
    } catch (e) {
      message.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  const breadcrumbItems = [
    { title: <HomeOutlined onClick={() => fetchPcapDirs('')} /> },
    ...pcapPath.split('/').filter(Boolean).map((part, i) => ({
      title: <a onClick={() => navigateBreadcrumb(i)}>{part}</a>,
    })),
  ];

  return (
    <Card title="Create DNS Send Task">
      <Form form={form} layout="vertical" onFinish={handleSubmit}
        onValuesChange={(changed) => {
          if (changed.input_type !== undefined) {
            setInputType(changed.input_type);
          }
        }}
      >
        <Form.Item name="name" label="Task Name" rules={[{ required: true }]}>
          <Input placeholder="My DNS Task" />
        </Form.Item>

        <Form.Item name="input_type" label="Input Type" rules={[{ required: true }]} initialValue="csv">
          <Select>
            <Select.Option value="csv">CSV (Domain List)</Select.Option>
            <Select.Option value="pcap">PCAP (Packet File)</Select.Option>
          </Select>
        </Form.Item>

        {inputType === 'csv' && (
          <Form.Item label="Upload File">
            <Upload beforeUpload={f => { setFile(f); return false; }} accept=".csv">
              <Button>{file ? file.name : 'Click to Upload'}</Button>
            </Upload>
          </Form.Item>
        )}

        {inputType === 'pcap' && (
          <Form.Item label="PCAP Directory" required>
            <Card size="small" style={{ marginBottom: 8 }}>
              <Breadcrumb items={breadcrumbItems} style={{ marginBottom: 12 }} />
              {pcapLoading ? (
                <Spin />
              ) : pcapData ? (
                <div>
                  {pcapData.dirs.length === 0 && pcapData.files.length === 0 && (
                    <Text type="secondary">Empty directory</Text>
                  )}
                  {pcapData.dirs.length > 0 && (
                    <List
                      header={<Text strong>Directories</Text>}
                      dataSource={pcapData.dirs}
                      renderItem={(d: string) => (
                        <List.Item
                          style={{ cursor: 'pointer' }}
                          onClick={() => navigateTo(d)}
                        >
                          <FolderOutlined style={{ marginRight: 8, color: '#1677ff' }} />
                          {d}
                        </List.Item>
                      )}
                    />
                  )}
                  {pcapData.files.length > 0 && (
                    <List
                      header={<Text strong>PCAP Files (click to select)</Text>}
                      dataSource={pcapData.files}
                      renderItem={(f: string) => {
                        const filePath = pcapPath ? `${pcapPath}/${f}` : f;
                        const isSelected = pcapPath.endsWith(f);
                        return (
                          <List.Item
                            style={{ cursor: 'pointer', background: isSelected ? '#e6f4ff' : undefined }}
                            onClick={() => { setPcapPath(filePath); }}
                          >
                            <FileOutlined style={{ marginRight: 8, color: isSelected ? '#1677ff' : '#52c41a' }} />
                            <Text style={{ color: isSelected ? '#1677ff' : undefined }}>{f}</Text>
                          </List.Item>
                        );
                      }}
                    />
                  )}
                </div>
              ) : (
                <Text type="secondary">Click to browse</Text>
              )}
            </Card>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Text type="secondary">
                Selected: {pcapPath || '(root — empty select a file)'}
              </Text>
              <Input
                placeholder="Or type path manually, e.g. test.pcap"
                value={pcapPath}
                onChange={e => setPcapPath(e.target.value)}
                allowClear
                size="small"
              />
            </Space>
          </Form.Item>
        )}

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
