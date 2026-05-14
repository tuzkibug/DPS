import { create } from 'zustand';
import type { Task, TaskStats, UpdateTaskRequest } from '../api/types';
import { taskAPI } from '../api';

interface TaskState {
  tasks: Task[];
  currentTask: Task | null;
  stats: Record<string, TaskStats>;
  qpsSamples: Record<string, Array<{ qps: number; time: number }>>;
  loading: boolean;
  error: string | null;

  fetchTasks: () => Promise<void>;
  fetchTask: (id: string) => Promise<void>;
  createTask: (data: Parameters<typeof taskAPI.create>[0]) => Promise<Task>;
  updateTask: (id: string, data: UpdateTaskRequest) => Promise<void>;
  deleteTask: (id: string) => Promise<void>;
  startTask: (id: string) => Promise<void>;
  stopTask: (id: string) => Promise<void>;
  batchStart: (ids: string[]) => Promise<void>;
  batchStop: (ids: string[]) => Promise<void>;
  batchDelete: (ids: string[]) => Promise<void>;
  setStats: (taskId: string, stats: TaskStats) => void;
  pushQpsSample: (taskId: string, qps: number, time: number) => void;
  setTaskStatus: (taskId: string, status: string) => void;
}

export const useTaskStore = create<TaskState>((set) => ({
  tasks: [],
  currentTask: null,
  stats: {},
  qpsSamples: {},
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

  batchStart: async (ids) => {
    for (const id of ids) {
      await taskAPI.start(id);
    }
    set(state => ({
      tasks: state.tasks.map(t => ids.includes(t.id) ? { ...t, status: 'running' as const } : t),
    }));
  },

  batchStop: async (ids) => {
    for (const id of ids) {
      await taskAPI.stop(id);
    }
    set(state => ({
      tasks: state.tasks.map(t => ids.includes(t.id) ? { ...t, status: 'pending' as const } : t),
    }));
  },

  batchDelete: async (ids) => {
    for (const id of ids) {
      try {
        await taskAPI.delete(id);
      } catch (_) { /* continue */ }
    }
    set(state => ({
      tasks: state.tasks.filter(t => !ids.includes(t.id)),
    }));
  },

  setStats: (taskId, stats) => {
    set(state => ({
      stats: { ...state.stats, [taskId]: stats },
    }));
  },

  pushQpsSample: (taskId, qps, time) => {
    set(state => {
      const prev = state.qpsSamples[taskId] ?? [];
      const next = [...prev, { qps, time }].slice(-60);
      return { qpsSamples: { ...state.qpsSamples, [taskId]: next } };
    });
  },

  setTaskStatus: (taskId, status) => {
    set(state => ({
      tasks: state.tasks.map(t => t.id === taskId ? { ...t, status: status as Task['status'] } : t),
      currentTask: state.currentTask?.id === taskId ? { ...state.currentTask, status: status as Task['status'] } : state.currentTask,
    }));
  },
}));
