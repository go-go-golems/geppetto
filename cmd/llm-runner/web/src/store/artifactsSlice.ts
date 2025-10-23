import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';
import type { ArtifactRun, ParsedRun } from '../types';
import { api } from '../api/client';

interface ArtifactsState {
  runs: ArtifactRun[];
  selectedRun: ParsedRun | null;
  loading: boolean;
  error: string | null;
}

const initialState: ArtifactsState = {
  runs: [],
  selectedRun: null,
  loading: false,
  error: null,
};

export const fetchRuns = createAsyncThunk('artifacts/fetchRuns', async () => {
  return await api.getRuns();
});

export const fetchRun = createAsyncThunk(
  'artifacts/fetchRun',
  async (runId: string) => {
    return await api.getRun(runId);
  }
);

const artifactsSlice = createSlice({
  name: 'artifacts',
  initialState,
  reducers: {
    clearSelectedRun: (state) => {
      state.selectedRun = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchRuns.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchRuns.fulfilled, (state, action: PayloadAction<ArtifactRun[]>) => {
        state.loading = false;
        state.runs = action.payload;
      })
      .addCase(fetchRuns.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch runs';
      })
      .addCase(fetchRun.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchRun.fulfilled, (state, action: PayloadAction<ParsedRun>) => {
        state.loading = false;
        state.selectedRun = action.payload;
      })
      .addCase(fetchRun.rejected, (state, action) => {
        state.loading = false;
        state.error = action.error.message || 'Failed to fetch run';
      });
  },
});

export const { clearSelectedRun } = artifactsSlice.actions;
export default artifactsSlice.reducer;

