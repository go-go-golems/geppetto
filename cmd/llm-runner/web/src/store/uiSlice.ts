import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

interface UIState {
  selectedTurnIndex: number;
  selectedEventSet: number;
  logFilter: string;
  logLevelFilter: string[];
  showRawData: boolean;
}

const initialState: UIState = {
  selectedTurnIndex: 0,
  selectedEventSet: 0,
  logFilter: '',
  logLevelFilter: [],
  showRawData: false,
};

const uiSlice = createSlice({
  name: 'ui',
  initialState,
  reducers: {
    setSelectedTurnIndex: (state, action: PayloadAction<number>) => {
      state.selectedTurnIndex = action.payload;
    },
    setSelectedEventSet: (state, action: PayloadAction<number>) => {
      state.selectedEventSet = action.payload;
    },
    setLogFilter: (state, action: PayloadAction<string>) => {
      state.logFilter = action.payload;
    },
    toggleLogLevel: (state, action: PayloadAction<string>) => {
      const level = action.payload;
      const index = state.logLevelFilter.indexOf(level);
      if (index > -1) {
        state.logLevelFilter.splice(index, 1);
      } else {
        state.logLevelFilter.push(level);
      }
    },
    toggleRawData: (state) => {
      state.showRawData = !state.showRawData;
    },
  },
});

export const {
  setSelectedTurnIndex,
  setSelectedEventSet,
  setLogFilter,
  toggleLogLevel,
  toggleRawData,
} = uiSlice.actions;
export default uiSlice.reducer;

