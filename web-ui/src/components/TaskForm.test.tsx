import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

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

const mockCreateTask = vi.hoisted(() => vi.fn());

vi.mock('../api', () => ({
  taskAPI: mockTaskAPI,
}));

vi.mock('../stores/taskStore', () => ({
  useTaskStore: (selector: (s: unknown) => unknown) => {
    const store = {
      tasks: [],
      currentTask: null,
      stats: {},
      loading: false,
      error: null,
      createTask: mockCreateTask,
      fetchTasks: vi.fn(),
      fetchTask: vi.fn(),
      updateTask: vi.fn(),
      deleteTask: vi.fn(),
      startTask: vi.fn(),
      stopTask: vi.fn(),
      setStats: vi.fn(),
    };
    return selector(store);
  },
}));

import { TaskForm } from './TaskForm';

describe('TaskForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTaskAPI.listPcapDirs.mockResolvedValue({
      dirs: [],
      files: ['test.pcap'],
      current_path: '/pcap',
    });
  });

  it('renders the create form', () => {
    render(<TaskForm />);

    expect(screen.getByText('Create DNS Send Task')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /create task/i })).toBeInTheDocument();
  });

  it('renders required fields', () => {
    render(<TaskForm />);

    expect(screen.getByLabelText(/task name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/source ip/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/destination ip/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/source mac/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/destination mac/i)).toBeInTheDocument();
  });

  it('validates required name field', async () => {
    render(<TaskForm />);

    const button = screen.getByRole('button', { name: /create task/i });
    await userEvent.click(button);

    expect(mockCreateTask).not.toHaveBeenCalled();
  });
});
