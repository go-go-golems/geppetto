import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Provider } from 'react-redux';
import { store } from './store';
import RunList from './components/RunList';
import RunView from './components/RunView';
import './App.css';

function App() {
  return (
    <Provider store={store}>
      <BrowserRouter>
        <div className="min-h-screen bg-gray-900 text-gray-100">
          <nav className="bg-gray-800 border-b border-gray-700">
            <div className="max-w-screen-2xl mx-auto px-4 py-3">
              <h1 className="text-xl font-bold text-white">
                LLM Runner Artifact Viewer
              </h1>
            </div>
          </nav>

          <Routes>
            <Route path="/" element={<RunList />} />
            <Route path="/run/:runId" element={<RunView />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </div>
      </BrowserRouter>
    </Provider>
  );
}

export default App;
