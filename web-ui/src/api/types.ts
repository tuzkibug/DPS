export interface Task {
  id: string;
  name: string;
  input_type: 'csv' | 'pcap';
  file_path: string;
  src_ip: string;
  dst_ip: string;
  src_mac: string;
  dst_mac: string;
  start_time: string;
  duration_ms: number;
  qos: QoSConfig;
  status: TaskStatus;
  created_at: string;
  updated_at: string;
  last_run_at: string | null;
  total_run_ms: number;
}

export type TaskStatus = 'pending' | 'running' | 'stopped' | 'completed' | 'failed';

export interface QoSConfig {
  target_qps: number;
  jitter: number;
  delay_min_ms: number;
  delay_max_ms: number;
}

export interface TaskStats {
  task_id: string;
  sent_count: number;
  failed_count: number;
  current_qps: number;
  start_time: string;
  elapsed_ms: number;
  status: TaskStatus;
  created_at: string;
  last_run_at: string | null;
  total_run_ms: number;
}

export interface CreateTaskRequest {
  name: string;
  input_type: 'csv' | 'pcap';
  file_content?: string;
  file_path?: string;
  src_ip: string;
  dst_ip: string;
  src_mac: string;
  dst_mac: string;
  start_time?: string;
  duration_ms?: number;
  qos: QoSConfig;
}

export interface PcapDirList {
  dirs: string[];
  files: string[];
  current_path: string;
}

export interface UpdateTaskRequest {
  name?: string;
  src_ip?: string;
  dst_ip?: string;
  src_mac?: string;
  dst_mac?: string;
  start_time?: string;
  duration_ms?: number;
  qos?: QoSConfig;
}