import axios from 'axios';
import type { ArtifactRun, ParsedRun } from '../types';

const apiClient = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

export const api = {
  getRuns: async (): Promise<ArtifactRun[]> => {
    const response = await apiClient.get<ArtifactRun[]>('/runs');
    return response.data;
  },

  getRun: async (runId: string): Promise<ParsedRun> => {
    const response = await apiClient.get<ParsedRun>(`/runs/${runId}`);
    return response.data;
  },
};

