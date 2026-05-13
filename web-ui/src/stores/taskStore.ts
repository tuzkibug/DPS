import { create } from 'zustand';
import type { Task, TaskStats, UpdateTaskRequest } from '../api/types';
import { taskAPI } from '../api';

interface TaskState {
  tasks: Task[];
  currentTask: Task | null;
  stats: Record<string, TaskStats>;
  loading: boolean;
  error: string | null;

  fetchTasks: () => Promise<void>;
  fetchTask: (id: string) => Promise<void>;
  createTask: (data: Parameters<typeof taskAPI.create>[0]) => Promise<Task>;
  updateTask: (id: string, data: UpdateTaskRequest) => Promise<void>;
  deleteTask: (id: string) => Promise<void>;
  startTask: (id: string) => Promise<void>;
  stopTask: (id: string) => Promise<void>;
  setStats: (taskId: string, stats: TaskStats) => void;
}

export const useTaskStore = create<TaskState>((set) => ({
  tasks: [],
  currentTask: null,
  stats: {},
  loading: false,
  error: null,

  fetchTasks: async () => {
    set({ loading: true, error: null });
    try {
      const tasks = await taskAPI.list();
      set({ tasks: tasks ?? [], loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  fetchTask: async (id: string) => {
    set({ loading: true, error: null });
    try {
      const task = await taskAPI.get(id);
      set({ currentTask: task, loading: false });
    } catch (e) {
      set({ error: (e as Error).message, loading: false });
    }
  },

  createTask: async (data) => {
    const task = await taskAPI.create(data);
    set(state => ({ tasks: [task, ...state.tasks] }));
    return task;
  },

  updateTask: async (id, data) => {
    const task = await taskAPI.update(id, data);
    set(state => ({
      tasks: state.tasks.map(t => t.id === id ? task : t),
      currentTask: state.currentTask?.id === id ? task : state.currentTask,
    }));
  },

  deleteTask: async (id) => {
    try {
      await taskAPI.delete(id);
      set(state => ({
        tasks: state.tasks.filter(t => t.id !== id),
        currentTask: state.currentTask?.id === id ? null : state.currentTask,
      }));
    } catch (e) {
      set({ error: (e as Error).message });
    }
  },

  startTask: async (id) => {
    await taskAPI.start(id);
    set(state => ({
      tasks: state.tasks.map(t => t.id === id ? { ...t, status: 'running' as const } : t),
      currentTask: state.currentTask?.id === id ? { ...state.currentTask, status: 'running' as const } : state.currentTask,
    }));
  },

  stopTask: async (id) => {
    await taskAPI.stop(id);
    set(state => ({
      tasks: state.tasks.map(t => t.id === id ? { ...t, status: 'pending' as const } : t),
      currentTask: state.currentTask?.id === id ? { ...state.currentTask, status: 'pending' as const } : state.currentTask,
    }));
  },

  setStats: (taskId, stats) => {
    set(state => ({
      stats: { ...state.stats, [taskId]: stats },
    }));
  },
}));
