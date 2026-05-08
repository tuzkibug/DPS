import axios from 'axios';
import type { Task, TaskStats, CreateTaskRequest, UpdateTaskRequest } from './types';

const api = axios.create({
  baseURL: '/api/v1',
});

export const taskAPI = {
  list: () => api.get<Task[]>('/tasks').then(r => r.data),

  get: (id: string) => api.get<Task>(`/tasks/${id}`).then(r => r.data),

  create: (data: CreateTaskRequest) => api.post<Task>('/tasks', data).then(r => r.data),

  update: (id: string, data: UpdateTaskRequest) => api.put<Task>(`/tasks/${id}`, data).then(r => r.data),

  delete: (id: string) => api.delete(`/tasks/${id}`),

  start: (id: string) => api.post(`/tasks/${id}/start`),

  stop: (id: string) => api.post(`/tasks/${id}/stop`),

  getStats: (id: string) => api.get<TaskStats>(`/tasks/${id}/stats`).then(r => r.data),

  getStatus: (id: string) => api.get<{ status: string }>(`/tasks/${id}/status`).then(r => r.data),
};

export const WS_URL = `ws://${window.location.host}/api/v1/ws/tasks`;

export default api;