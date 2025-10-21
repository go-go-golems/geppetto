import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAppDispatch, useAppSelector } from '../store/hooks';
import { fetchRuns } from '../store/artifactsSlice';
import { Clock, FileText, AlertCircle } from 'lucide-react';

export default function RunList() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { runs, loading, error } = useAppSelector((state) => state.artifacts);

  useEffect(() => {
    dispatch(fetchRuns());
  }, [dispatch]);

  if (loading && runs.length === 0) {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-gray-400">Loading runs...</div>
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

  return (
    <div className="max-w-screen-xl mx-auto px-4 py-8">
      <h2 className="text-2xl font-bold mb-6">Artifact Runs</h2>

      {runs.length === 0 ? (
        <div className="text-center text-gray-400 py-12">
          <FileText size={48} className="mx-auto mb-4 opacity-50" />
          <p>No runs found</p>
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {runs.map((run) => (
            <div
              key={run.id}
              onClick={() => navigate(`/run/${run.id}`)}
              className="bg-gray-800 border border-gray-700 rounded-lg p-4 cursor-pointer hover:bg-gray-750 hover:border-gray-600 transition-colors"
            >
              <h3 className="text-lg font-semibold mb-2 text-white">{run.id}</h3>
              
              <div className="flex items-center text-sm text-gray-400 mb-3">
                <Clock size={14} className="mr-1" />
                {new Date(run.timestamp * 1000).toLocaleString()}
              </div>

              <div className="flex flex-wrap gap-2 mb-3">
                {run.hasTurns && (
                  <span className="px-2 py-1 text-xs bg-blue-900 text-blue-200 rounded">
                    {run.turnCount} turn{run.turnCount !== 1 ? 's' : ''}
                  </span>
                )}
                {run.hasEvents && (
                  <span className="px-2 py-1 text-xs bg-green-900 text-green-200 rounded">
                    Events
                  </span>
                )}
                {run.hasLogs && (
                  <span className="px-2 py-1 text-xs bg-purple-900 text-purple-200 rounded">
                    Logs
                  </span>
                )}
                {run.hasRaw && (
                  <span className="px-2 py-1 text-xs bg-orange-900 text-orange-200 rounded">
                    Raw Data
                  </span>
                )}
              </div>

              <div className="text-sm text-gray-500">
                Click to view details
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

