import { useEffect, useState } from 'react';
import { useParams, useNavigate, useSearchParams } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { fetchRun } from '../store/artifactsSlice';
import { ArrowLeft, AlertCircle } from 'lucide-react';
import TurnTimeline from './TurnTimeline';
import EventStream from './EventStream';
import LogViewer from './LogViewer';
import RawDataInspector from './RawDataInspector';
import ErrorContext from './ErrorContext';

export type TabType = 'timeline' | 'events' | 'logs' | 'raw' | 'errors';

export default function RunView() {
  const { runId } = useParams<{ runId: string }>();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const { selectedRun, loading, error } = useAppSelector((state) => state.artifacts);
  const [searchParams, setSearchParams] = useSearchParams();
  const [activeTab, setActiveTab] = useState<TabType>((searchParams.get('tab') as TabType) || 'timeline');

  useEffect(() => {
    if (runId) {
      dispatch(fetchRun(runId));
    }
  }, [dispatch, runId]);

  useEffect(() => {
    const tab = searchParams.get('tab') as TabType;
    if (tab) {
      setActiveTab(tab);
    }
  }, [searchParams]);

  const navigateToTab = (tab: TabType, turnIndex?: number) => {
    setActiveTab(tab);
    const params = new URLSearchParams({ tab });
    if (turnIndex !== undefined) {
      params.set('turn', turnIndex.toString());
    }
    setSearchParams(params);
  };

  if (loading && !selectedRun) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-gray-400">Loading run data...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-red-400 flex items-center gap-2">
          <AlertCircle size={20} />
          {error}
        </div>
      </div>
    );
  }

  if (!selectedRun) {
    return null;
  }

  const tabs: { id: TabType; label: string; count?: number }[] = [
    { id: 'timeline', label: 'Turn Timeline', count: selectedRun.turns.length },
    { id: 'events', label: 'Events', count: selectedRun.events.reduce((sum, e) => sum + e.length, 0) },
    { id: 'logs', label: 'Logs', count: selectedRun.logs.length },
    { id: 'raw', label: 'Raw Data', count: selectedRun.raw.length },
    { id: 'errors', label: 'Errors', count: selectedRun.errors.length },
  ];

  return (
    <div className="h-screen flex flex-col">
      {/* Header */}
      <div className="bg-gray-800 border-b border-gray-700 px-4 py-3">
        <div className="max-w-screen-2xl mx-auto flex items-center justify-between">
          <div className="flex items-center gap-4">
            <button
              onClick={() => navigate('/')}
              className="text-gray-400 hover:text-white transition-colors"
            >
              <ArrowLeft size={20} />
            </button>
            <div>
              <h2 className="text-lg font-semibold text-white">{selectedRun.id}</h2>
              <div className="text-sm text-gray-400">
                {selectedRun.turns.length} turn{selectedRun.turns.length !== 1 ? 's' : ''} •{' '}
                {selectedRun.logs.length} log{selectedRun.logs.length !== 1 ? 's' : ''}
                {selectedRun.errors.length > 0 && (
                  <span className="ml-2 text-red-400">
                    • {selectedRun.errors.length} error{selectedRun.errors.length !== 1 ? 's' : ''}
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-gray-800 border-b border-gray-700">
        <div className="max-w-screen-2xl mx-auto px-4">
          <div className="flex gap-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => navigateToTab(tab.id)}
                className={`px-4 py-2 text-sm font-medium transition-colors relative ${
                  activeTab === tab.id
                    ? 'text-white bg-gray-700'
                    : 'text-gray-400 hover:text-white hover:bg-gray-750'
                }`}
              >
                {tab.label}
                {tab.count !== undefined && tab.count > 0 && (
                  <span className="ml-2 px-1.5 py-0.5 text-xs bg-gray-600 rounded">
                    {tab.count}
                  </span>
                )}
                {tab.id === 'errors' && tab.count! > 0 && (
                  <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
                )}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-hidden bg-gray-900">
        <div className="h-full max-w-screen-2xl mx-auto">
          {activeTab === 'timeline' && <TurnTimeline run={selectedRun} onNavigate={navigateToTab} />}
          {activeTab === 'events' && <EventStream run={selectedRun} onNavigate={navigateToTab} />}
          {activeTab === 'logs' && <LogViewer run={selectedRun} onNavigate={navigateToTab} />}
          {activeTab === 'raw' && <RawDataInspector run={selectedRun} onNavigate={navigateToTab} />}
          {activeTab === 'errors' && <ErrorContext run={selectedRun} onNavigate={navigateToTab} />}
        </div>
      </div>
    </div>
  );
}

