import { configureStore } from '@reduxjs/toolkit';
import artifactsReducer from './artifactsSlice';
import uiReducer from './uiSlice';

export const store = configureStore({
  reducer: {
    artifacts: artifactsReducer,
    ui: uiReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;

