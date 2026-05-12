import { describe, it, expect, vi, beforeEach } from 'vitest';

const mockTaskAPI = vi.hoisted(() => ({
  list: vi.fn(),
  get: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  start: vi.fn(),
  stop: vi.fn(),
  getStats: vi.fn(),
  getStatus: vi.fn(),
  listPcapDirs: vi.fn(),
}));

vi.mock('../api', () => ({
  taskAPI: mockTaskAPI,
}));

import { useTaskStore } from './taskStore';

function makeTask(id: string, overrides = {}) {
  return {
    id,
    name: 'test-task',
    input_type: 'csv' as const,
    file_path: '/tmp/test.csv',
    src_ip: '192.168.1.1',
    dst_ip: '8.8.8.8',
    src_mac: 'aa:bb:cc:dd:ee:ff',
    dst_mac: '11:22:33:44:55:66',
    start_time: '',
    duration_ms: 0,
    qos: { target_qps: 100, jitter: 0, delay_min_ms: 0, delay_max_ms: 0 },
    status: 'pending' as const,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
    last_run_at: null,
    total_run_ms: 0,
    ...overrides,
  };
}

describe('taskStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useTaskStore.setState({
      tasks: [],
      currentTask: null,
      stats: {},
      loading: false,
      error: null,
    });
  });

  describe('fetchTasks', () => {
    it('loads tasks and updates state', async () => {
      const tasks = [makeTask('1'), makeTask('2')];
      mockTaskAPI.list.mockResolvedValueOnce(tasks);

      await useTaskStore.getState().fetchTasks();

      expect(useTaskStore.getState().tasks).toEqual(tasks);
      expect(useTaskStore.getState().loading).toBe(false);
      expect(useTaskStore.getState().error).toBeNull();
    });

    it('sets error on failure', async () => {
      mockTaskAPI.list.mockRejectedValueOnce(new Error('network error'));

      await useTaskStore.getState().fetchTasks();

      expect(useTaskStore.getState().tasks).toEqual([]);
      expect(useTaskStore.getState().error).toBe('network error');
      expect(useTaskStore.getState().loading).toBe(false);
    });
  });

  describe('createTask', () => {
    it('prepends created task to list', async () => {
      const existing = makeTask('1');
      useTaskStore.setState({ tasks: [existing] });

      const newTask = makeTask('2', { name: 'new-task' });
      mockTaskAPI.create.mockResolvedValueOnce(newTask);

      const createData = {
        name: 'new-task',
        input_type: 'csv' as const,
        src_ip: '10.0.0.1',
        dst_ip: '10.0.0.2',
        src_mac: 'aa:bb:cc:dd:ee:ff',
        dst_mac: '11:22:33:44:55:66',
        qos: { target_qps: 100, jitter: 0, delay_min_ms: 0, delay_max_ms: 0 },
      };

      const result = await useTaskStore.getState().createTask(createData);

      expect(result).toEqual(newTask);
      expect(useTaskStore.getState().tasks).toHaveLength(2);
      expect(useTaskStore.getState().tasks[0]).toEqual(newTask);
    });
  });

  describe('deleteTask', () => {
    it('removes task from list', async () => {
      useTaskStore.setState({ tasks: [makeTask('1'), makeTask('2')] });

      await useTaskStore.getState().deleteTask('1');

      expect(useTaskStore.getState().tasks).toHaveLength(1);
      expect(useTaskStore.getState().tasks[0].id).toBe('2');
    });

    it('clears currentTask if deleted', async () => {
      useTaskStore.setState({
        tasks: [makeTask('1')],
        currentTask: makeTask('1'),
      });

      await useTaskStore.getState().deleteTask('1');

      expect(useTaskStore.getState().currentTask).toBeNull();
    });
  });

  describe('startTask / stopTask', () => {
    it('optimistically sets status to running on start', async () => {
      useTaskStore.setState({ tasks: [makeTask('1')] });

      await useTaskStore.getState().startTask('1');

      expect(useTaskStore.getState().tasks[0].status).toBe('running');
    });

    it('optimistically sets status to pending on stop', async () => {
      useTaskStore.setState({ tasks: [makeTask('1', { status: 'running' })] });

      await useTaskStore.getState().stopTask('1');

      expect(useTaskStore.getState().tasks[0].status).toBe('pending');
    });
  });

  describe('setStats', () => {
    it('updates stats for a task', () => {
      useTaskStore.getState().setStats('1', {
        task_id: '1',
        sent_count: 42,
        failed_count: 0,
        current_qps: 100,
        start_time: '2026-01-01T00:00:00Z',
        elapsed_ms: 5000,
        status: 'running',
        created_at: '2026-01-01T00:00:00Z',
        last_run_at: null,
        total_run_ms: 0,
      });

      expect(useTaskStore.getState().stats['1']).toBeDefined();
      expect(useTaskStore.getState().stats['1'].sent_count).toBe(42);
    });
  });
});
